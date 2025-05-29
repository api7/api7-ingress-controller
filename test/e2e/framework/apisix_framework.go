// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_apisixNamespace = "apisix-standalone-e2e"
	_apisixFramework *APISIXFramework
)

// APISIXFramework implements TestFramework for APISIX standalone
type APISIXFramework struct {
	Context context.Context
	GinkgoT GinkgoTInterface
	GomegaT *GomegaWithT

	Logger logger.TestLogger

	kubectlOpts *k8s.KubectlOptions
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config
	K8sClient   client.Client
	namespace   string
}

// NewAPISIXFramework creates a new APISIX framework
func NewAPISIXFramework() *APISIXFramework {
	f := &APISIXFramework{
		GinkgoT: GinkgoT(),
		GomegaT: NewWithT(GinkgoT(4)),
		Logger:  logger.Terratest,
	}

	f.Context = context.TODO()

	// Use environment variable for namespace if set
	namespace := os.Getenv("APISIX_NAMESPACE")
	if namespace == "" {
		namespace = _apisixNamespace
	}

	f.namespace = namespace

	f.kubectlOpts = k8s.NewKubectlOptions("", "", namespace)
	restCfg, err := buildRestConfig("")
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "building API Server rest config")
	f.restConfig = restCfg

	clientset, err := kubernetes.NewForConfig(restCfg)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating Kubernetes clientset")
	f.clientset = clientset

	k8sClient, err := client.New(restCfg, client.Options{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating controller-runtime client")
	f.K8sClient = k8sClient

	_apisixFramework = f

	return f
}

// BeforeSuite initializes the APISIX test environment
func (f *APISIXFramework) BeforeSuite() {
	f.Logf("Starting APISIX standalone test suite")
	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, f.namespace)

	Eventually(func() error {
		_, err := k8s.GetNamespaceE(GinkgoT(), f.kubectlOpts, f.namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("namespace %s still exists", f.namespace)
	}, "1m", "2s").Should(Succeed())

	k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, f.namespace)

	f.Logf("APISIX standalone test environment initialized")
}

// AfterSuite cleans up the APISIX test environment
func (f *APISIXFramework) AfterSuite() {
	f.Logf("Cleaning up APISIX standalone test environment")

	// Clean up namespace
	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, f.kubectlOpts.Namespace)
}

// GetFramework returns the global APISIX framework instance
func GetAPISIXFramework() *APISIXFramework {
	return _apisixFramework
}

// Logf logs a formatted message
func (f *APISIXFramework) Logf(format string, v ...any) {
	f.Logger.Logf(f.GinkgoT, format, v...)
}

func (f *APISIXFramework) DeployIngress(opts IngressDeployOpts) {
	buf := bytes.NewBuffer(nil)

	err := IngressSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering ingress spec")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err = WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "waiting for controller-manager pod ready")
	f.WaitControllerManagerLog("All cache synced successfully", 0, time.Minute)
}

func (f *APISIXFramework) WaitControllerManagerLog(keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog("control-plane=controller-manager", keyword, sinceSeconds, timeout)
}

func (f *APISIXFramework) WaitDPLog(keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog("app.kubernetes.io/name=apisix", keyword, sinceSeconds, timeout)
}

func (f *APISIXFramework) WaitPodsLog(selector, keyword string, sinceSeconds int64, timeout time.Duration) {
	pods := f.ListRunningPods(selector)
	wg := sync.WaitGroup{}
	for _, p := range pods {
		wg.Add(1)
		go func(p corev1.Pod) {
			defer wg.Done()
			opts := corev1.PodLogOptions{Follow: true}
			if sinceSeconds > 0 {
				opts.SinceSeconds = ptr.To(sinceSeconds)
			} else {
				opts.TailLines = ptr.To(int64(0))
			}
			logStream, err := f.clientset.CoreV1().Pods(p.Namespace).GetLogs(p.Name, &opts).Stream(context.Background())
			f.GomegaT.Expect(err).Should(gomega.BeNil())
			scanner := bufio.NewScanner(logStream)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, keyword) {
					return
				}
			}
		}(p)
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		f.GinkgoT.Error("wait log timeout")
	}
}

func (f *APISIXFramework) ListRunningPods(selector string) []corev1.Pod {
	pods, err := f.clientset.CoreV1().Pods(f.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "list pod: ", selector)
	runningPods := make([]corev1.Pod, 0)
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			runningPods = append(runningPods, p)
		}
	}
	return runningPods
}
