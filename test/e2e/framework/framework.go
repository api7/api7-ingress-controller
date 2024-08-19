package framework

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	_namespace = "api7-ee-e2e"
	_framework *Framework
)
var (
	//go:embed manifests/license.key
	privateKey string
)

type DataPlanePod struct {
	Selector string
	PodName  string
}

type DataPlaneContext struct {
	Context    context.Context
	CancelFunc context.CancelFunc
}

type Framework struct {
	Context context.Context
	GinkgoT GinkgoTInterface
	GomegaT *GomegaWithT

	Logger logger.TestLogger

	kubectlOpts *k8s.KubectlOptions
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config

	DB         *gorm.DB
	RawETCD    *clientv3.Client
	PrivateKey *rsa.PrivateKey

	License      string
	BuiltInRoles map[string]string

	Revision          int64
	dpLogChan         map[DataPlanePod]chan string
	dpLogWatchContext map[string]*DataPlaneContext

	dashboardHTTPTunnel  *k8s.Tunnel
	dashboardHTTPSTunnel *k8s.Tunnel
}

// NewFramework create a global framework with special settings.
func NewFramework() *Framework {
	GinkgoWriter.Println("into NewFramework")

	f := &Framework{
		GinkgoT:           GinkgoT(),
		GomegaT:           NewWithT(GinkgoT(4)),
		BuiltInRoles:      make(map[string]string),
		dpLogChan:         make(map[DataPlanePod]chan string),
		dpLogWatchContext: make(map[string]*DataPlaneContext),
		Logger:            logger.Terratest,
	}

	// FIXME if we need some precise control on the context
	f.Context = context.TODO()

	f.kubectlOpts = k8s.NewKubectlOptions("", "", _namespace)
	restCfg, err := buildRestConfig("")
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "building API Server rest config")
	f.restConfig = restCfg

	clientset, err := kubernetes.NewForConfig(restCfg)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating Kubernetes clientset")
	f.clientset = clientset

	block, _ := pem.Decode([]byte(privateKey))
	//nolint:all
	if block == nil {
		f.GomegaT.Fail("failed to decode private key")
	}
	//nolint:all
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		f.GomegaT.Fail("failed to parse private key")
	}
	f.PrivateKey = key

	_framework = f

	BeforeSuite(func() {
		_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)

		Eventually(func() error {
			_, err := k8s.GetNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)
			if k8serrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("namespace %s still exists", _namespace)
		}, "1m", "2s").Should(Succeed())

		k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, _namespace)

		f.DeployComponents()

		time.Sleep(1 * time.Minute)
		err := f.newDashboardTunnel()
		f.Logger.Logf(GinkgoT(), "Dashboard HTTP Tunnel:"+f.dashboardHTTPTunnel.Endpoint())
		Expect(err).ShouldNot(HaveOccurred(), "creating dashboard tunnel")

		f.UploadLicense()

		f.setDpManagerEndpoints()
	})
	AfterSuite(func() {
		f.shutdownDashboardTunnel()
		//cmd := exec.Command("kubectl", "delete", "ns", _namespace)
		//_, _ = utils.Run(cmd)
	})

	GinkgoWriter.Println("Another debug message")

	return f
}

type Items[T any] []T

func (f *Framework) BatchDeletePublishedService(serviceIDs Items[string]) {
}
func GetFramework() *Framework {
	return _framework
}

func (f *Framework) Base64Encode(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}

// DeployComponents deploy necessary components
func (f *Framework) DeployComponents() {
	f.deployMySQL()
	f.deployMockS3()
	f.deploy()
	//f.deployHTTPBIN()
	//f.deployNginx()
	f.initDashboard()
}

func (f *Framework) setDpManagerEndpoints() {
	payload := []byte(fmt.Sprintf(`{"control_plane_address":["%s"]}`, DPManagerTLSEndpoint))

	respExp := f.DashboardHTTPClient().
		PUT("/api/system_settings").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payload).
		Expect()

	respExp.Raw()
	f.Logger.Logf(f.GinkgoT, "set dp manager endpoints response: %s", respExp.Body().Raw())

	respExp.Status(200).
		Body().Contains("control_plane_address")
}
