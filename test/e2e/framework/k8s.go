package framework

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

// buildRestConfig builds the rest.Config object from kubeconfig filepath and
// context, if kubeconfig is missing, building from in-cluster configuration.
func buildRestConfig(context string) (*rest.Config, error) {

	// Config loading rules:
	// 1. kubeconfig if it not empty string
	// 2. Config(s) in KUBECONFIG environment variable
	// 3. In cluster config if running in-cluster
	// 4. Use $HOME/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}

func (f *Framework) ensureService(name, namespace string, desiredEndpoints int) error {
	return f.ensureServiceWithTimeout(name, namespace, desiredEndpoints, 120)
}

func (f *Framework) ensureServiceWithTimeout(name, namespace string, desiredEndpoints, timeout int) error {
	backoff := wait.Backoff{
		Duration: 6 * time.Second,
		Factor:   1,
		Steps:    timeout / 6,
	}
	var lastErr error
	condFunc := func() (bool, error) {
		ep, err := f.clientset.CoreV1().Endpoints(namespace).Get(f.Context, name, metav1.GetOptions{})
		if err != nil {
			lastErr = err
			log.Errorw("failed to list endpoints",
				zap.String("service", name),
				zap.Error(err),
			)
			return false, nil
		}
		count := 0
		for _, ss := range ep.Subsets {
			count += len(ss.Addresses)
		}
		if count == desiredEndpoints {
			return true, nil
		}
		log.Infow("endpoints count mismatch",
			zap.String("service", name),
			zap.Any("ep", ep),
			zap.Int("expected", desiredEndpoints),
			zap.Int("actual", count),
		)
		lastErr = fmt.Errorf("expected endpoints: %d but seen %d", desiredEndpoints, count)
		return false, nil
	}

	err := wait.ExponentialBackoff(backoff, condFunc)
	if err != nil {
		return lastErr
	}
	return nil
}

func (f *Framework) GetServiceEndpoints(name string) ([]string, error) {
	ep, err := f.clientset.CoreV1().Endpoints(_namespace).Get(f.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var endpoints []string
	for _, ss := range ep.Subsets {
		for _, addr := range ss.Addresses {
			endpoints = append(endpoints, addr.IP)
		}
	}
	return endpoints, nil
}

func (f *Framework) deletePods(selector string) {
	podList, err := f.clientset.CoreV1().Pods(_namespace).List(f.Context, metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "list pods")
	for _, pod := range podList.Items {
		_ = f.clientset.CoreV1().
			Pods(_namespace).
			Delete(f.Context, pod.Name, metav1.DeleteOptions{GracePeriodSeconds: ptr.To(int64(30))})
	}
}

func (f *Framework) CreateNamespaceWithTestService(name string) {
	_, err := f.clientset.CoreV1().
		Namespaces().
		Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "create namespace")
		return
	}

	_, err = f.clientset.CoreV1().Services(name).Create(f.Context, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "httpbin",
			},
			Type: v1.ServiceTypeClusterIP,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "create service")
	}
}

func (f *Framework) DeleteNamespace(name string) {
	err := f.clientset.CoreV1().Namespaces().Delete(f.Context, name, metav1.DeleteOptions{})
	if err == nil || errors.IsNotFound(err) {
		return
	}
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "delete namespace")
}

func (f *Framework) Scale(name string, replicas int32) {
	scale, err := f.clientset.AppsV1().Deployments(_namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("get deployment %s scale failed", name))
	if scale.Spec.Replicas == replicas {
		return
	}
	scale.Spec.Replicas = replicas
	_, err = f.clientset.AppsV1().
		Deployments(_namespace).
		UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("scale deployment %s to %v failed", name, replicas))

	err = f.ensureService(name, _namespace, int(replicas))
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(),
		fmt.Sprintf("ensure service %s/%s has %v endpoints failed", _namespace, name, replicas))
}

func (f *Framework) GetPodIP(selector string) string {
	podList, err := f.clientset.CoreV1().Pods(_namespace).List(f.Context, metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred())
	f.GomegaT.Expect(podList.Items).ShouldNot(BeEmpty())
	return podList.Items[0].Status.PodIP
}

func (f *Framework) newDashboardTunnel() error {
	var (
		httpNodePort  int
		httpsNodePort int
		httpPort      int
		httpsPort     int
	)

	service := k8s.GetService(f.GinkgoT, f.kubectlOpts, "api7ee3-dashboard")

	for _, port := range service.Spec.Ports {
		if port.Name == "http" {
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		} else if port.Name == "https" {
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		}
	}

	f.dashboardHTTPTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpNodePort, httpPort)
	f.dashboardHTTPSTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpsNodePort, httpsPort)

	if err := f.dashboardHTTPTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}
	if err := f.dashboardHTTPSTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}

	return nil
}

func (f *Framework) shutdownDashboardTunnel() {
	if f.dashboardHTTPTunnel != nil {
		f.dashboardHTTPTunnel.Close()
	}
	if f.dashboardHTTPSTunnel != nil {
		f.dashboardHTTPSTunnel.Close()
	}
}

func (f *Framework) GetDashboardEndpoint() string {
	return f.dashboardHTTPTunnel.Endpoint()
}

func (f *Framework) GetDashboardEndpointHTTPS() string {
	return f.dashboardHTTPSTunnel.Endpoint()
}

func (f *Framework) DashboardHTTPClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   f.GetDashboardEndpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

func (f *Framework) DashboardHTTPSClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   f.GetDashboardEndpointHTTPS(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}
