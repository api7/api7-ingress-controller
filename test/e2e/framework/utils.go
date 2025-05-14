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
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
)

var (
	scheme_ = runtime.NewScheme()
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme_))
	if err := gatewayv1.Install(scheme_); err != nil {
		panic(err)
	}
	if err := v1alpha1.AddToScheme(scheme_); err != nil {
		panic(err)
	}
}

func (f *Framework) NewExpectResponse(httpBody any) *httpexpect.Response {
	body, err := json.Marshal(httpBody)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred())

	return httpexpect.NewResponse(f.GinkgoT, &http.Response{
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(body)),
	})
}

// ListPods query pods by label selector.
func (f *Framework) ListPods(selector string) []corev1.Pod {
	pods, err := f.clientset.CoreV1().Pods(_namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "list pod: ", selector)
	return pods.Items
}

func (f *Framework) ListRunningPods(selector string) []corev1.Pod {
	pods, err := f.clientset.CoreV1().Pods(_namespace).List(context.TODO(), metav1.ListOptions{
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

// ExecCommandInPod exec cmd in specify pod and return the output from stdout and stderr
func (f *Framework) ExecCommandInPod(podName string, cmd ...string) (string, string) {
	req := f.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(_namespace).SubResource("exec")
	req.VersionedParams(
		&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		},
		scheme.ParameterCodec,
	)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(f.restConfig, "POST", req.URL())
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "request kubernetes exec api")
	_ = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())
}

func (f *Framework) LoginDashboardBySAML(
	loginPath string,
	username, password string,
	redirectURI string,
	acsErrmsg string,
) ([]*http.Cookie, []*http.Cookie) {
	client := http.DefaultClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	u, err := url.Parse(DashboardEndpoint)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	u.Path = loginPath
	u.RawQuery = fmt.Sprintf("redirect_uri=%s", redirectURI)

	// 1: get location to keycloak
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	resp, err := client.Do(req)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "sending user login request")
	defer func() { _ = resp.Body.Close() }()

	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(302))
	location := resp.Header.Get("Location")
	f.GomegaT.Expect(location).Should(gomega.ContainSubstring("SAMLRequest"))
	consoleCookie := resp.Header.Values("Set-Cookie")

	// 2: request keycloak, keycloak return an html page
	req, err = http.NewRequest(http.MethodGet, location, nil)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	resp, err = client.Do(req)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "sending login page request")
	defer func() { _ = resp.Body.Close() }()
	keycloakCookie := resp.Header.Values("Set-Cookie")
	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(200))
	doc, err := html.Parse(resp.Body)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "parse login page html")

	HTMLMap := f.ParseHTML(doc)
	loginURL := HTMLMap["kc-form-login"]

	// 3: request keycloak login API, keycloak login success, and then return redirect HTML page
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	req, err = http.NewRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", strings.Join(keycloakCookie, ";"))

	resp, err = client.Do(req)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	defer func() { _ = resp.Body.Close() }()
	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(200))

	// save login keycloak cookies, for logout if we need
	keycloakLoginCookies := resp.Cookies()

	// 4: callback to dashboard acs URL
	doc, err = html.Parse(resp.Body)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	HTMLMap = f.ParseHTML(doc)
	acsURL := HTMLMap["saml-post-binding"]
	form = url.Values{}
	form.Add("SAMLResponse", HTMLMap["SAMLResponse"])
	form.Add("RelayState", HTMLMap["RelayState"])
	req, err = http.NewRequest(http.MethodPost, acsURL, strings.NewReader(form.Encode()))
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", strings.Join(consoleCookie, ";"))

	resp, err = client.Do(req)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	defer func() { _ = resp.Body.Close() }()
	if len(acsErrmsg) > 0 {
		f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(http.StatusSeeOther))
		f.GomegaT.Expect(err).Should(gomega.BeNil())
		f.GomegaT.Expect(resp.Header.Get("Location")).Should(gomega.ContainSubstring("/login?err_msg"))
		f.GomegaT.Expect(resp.Header.Get("Location")).Should(gomega.ContainSubstring(url.QueryEscape(acsErrmsg)))
		return nil, nil
	}

	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(302))
	location = resp.Header.Get("Location")
	u, err = url.Parse(location)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	f.GomegaT.Expect(u.EscapedPath()).Should(gomega.Equal(redirectURI))
	return resp.Cookies(), keycloakLoginCookies
}

func (f *Framework) LogoutDashboardBySAML(
	logoutPath string,
	cookies []*http.Cookie,
	keycloakCookies []*http.Cookie,
	redirectURI string,
	logoutIDPSession bool,
	logoutErrMsg string,
) {
	client := http.DefaultClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	u, err := url.Parse(DashboardEndpoint)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	u.Path = logoutPath
	u.RawQuery = fmt.Sprintf("redirect_uri=%s", redirectURI)

	// 1: request logout path, get location to keycloak
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	// set dashboard login cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	defer func() { _ = resp.Body.Close() }()
	if len(logoutErrMsg) > 0 {
		f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(http.StatusSeeOther))
		f.GomegaT.Expect(resp.Header.Get("Location")).Should(gomega.ContainSubstring("/login?err_msg"))
		f.GomegaT.Expect(resp.Header.Get("Location")).Should(gomega.ContainSubstring(url.QueryEscape(logoutErrMsg)))
		return
	}
	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(302))
	location := resp.Header.Get("Location")
	if !logoutIDPSession {
		f.GomegaT.Expect(location).Should(gomega.Equal(redirectURI))
		return
	}
	f.GomegaT.Expect(location).Should(gomega.ContainSubstring("SAMLRequest"))

	// 2: request keycloak, keycloak return an html page
	req, err = http.NewRequest(http.MethodGet, location, nil)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	// set keycloak login cookies
	for _, cookie := range keycloakCookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "sending login page request")
	defer func() { _ = resp.Body.Close() }()
	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(200))
	doc, err := html.Parse(resp.Body)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "parse login page html")

	// 3: request slo URL
	HTMLMap := f.ParseHTML(doc)
	sloURL := HTMLMap["saml-post-binding"]
	form := url.Values{}
	form.Add("SAMLResponse", HTMLMap["SAMLResponse"])
	form.Add("RelayState", HTMLMap["RelayState"])
	req, err = http.NewRequest(http.MethodPost, sloURL, strings.NewReader(form.Encode()))
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	defer func() { _ = resp.Body.Close() }()
	f.GomegaT.Expect(resp.StatusCode).Should(gomega.Equal(302))
	f.GomegaT.Expect(resp.Header.Get("Location")).Should(gomega.Equal(redirectURI))
}

func (f *Framework) LogoutDashboardByOIDC(logoutPath string, cookies []*http.Cookie) error {
	client := http.DefaultClient
	u, err := url.Parse(DashboardEndpoint)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred())
	u.Path = logoutPath

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	f.GomegaT.Expect(err).Should(gomega.BeNil())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "sending user logout request")
	defer func() {
		_ = resp.Body.Close()
	}()
	return nil
}

func (f *Framework) GetPodLogs(name string, previous bool) string {
	reader, err := f.clientset.CoreV1().
		Pods(_namespace).
		GetLogs(name, &corev1.PodLogOptions{Previous: previous}).
		Stream(context.Background())
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "get logs")
	defer func() {
		_ = reader.Close()
	}()

	logs, err := io.ReadAll(reader)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "read all logs")

	return string(logs)
}

func (f *Framework) WaitMTLSDPLog(keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog("app.kubernetes.io/name=apisix,cp-connection=mtls", keyword, sinceSeconds, timeout)
}

func (f *Framework) WaitControllerManagerLog(keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog("control-plane=controller-manager", keyword, sinceSeconds, timeout)
}

func (f *Framework) WaitDPLog(keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog("app.kubernetes.io/name=apisix", keyword, sinceSeconds, timeout)
}

func (f *Framework) WaitPodsLog(selector, keyword string, sinceSeconds int64, timeout time.Duration) {
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

func CreateTestZipFile(sourceCode, metadata string) ([]byte, error) {
	// Create a new zip file
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// Add files to the zip archive
	if err := addFileToZip(zipWriter, "plugin.lua", sourceCode); err != nil {
		return nil, err
	}
	if err := addFileToZip(zipWriter, "metadata.json", metadata); err != nil {
		return nil, err
	}

	// Close the zip writer
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return zipBuffer.Bytes(), nil
}

func HTTPRoutePolicyMustHaveCondition(t testing.TestingT, client client.Client, timeout time.Duration, refNN, hrpNN types.NamespacedName, condition metav1.Condition) {
	err := EventuallyHTTPRoutePolicyHaveStatus(client, timeout, hrpNN, func(httpRoutePolicy v1alpha1.HTTPRoutePolicy, status v1alpha1.PolicyStatus) bool {
		var (
			ancestors      = status.Ancestors
			conditionFound bool
		)
		for _, ancestor := range ancestors {
			if err := kubernetes.ConditionsHaveLatestObservedGeneration(&httpRoutePolicy, ancestor.Conditions); err != nil {
				log.Printf("HTTPRoutePolicy %s (parentRef=%v) %v", hrpNN, parentRefToString(ancestor.AncestorRef), err)
				return false
			}

			if ancestor.AncestorRef.Name == gatewayv1.ObjectName(refNN.Name) && (ancestor.AncestorRef.Namespace == nil || string(*ancestor.AncestorRef.Namespace) == refNN.Namespace) {
				if findConditionInList(t, ancestor.Conditions, condition) {
					conditionFound = true
				}
			}
		}
		return conditionFound
	})

	require.NoError(t, err, "error waiting for HTTPRoutePolicy status to have a Condition matching expectations")
}

func EventuallyHTTPRoutePolicyHaveStatus(client client.Client, timeout time.Duration, hrpNN types.NamespacedName, f func(httpRoutePolicy v1alpha1.HTTPRoutePolicy, status v1alpha1.PolicyStatus) bool) error {
	_ = v1alpha1.AddToScheme(client.Scheme())
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		var httpRoutePolicy v1alpha1.HTTPRoutePolicy
		if err = client.Get(ctx, hrpNN, &httpRoutePolicy); err != nil {
			return false, errors.Errorf("error fetching HTTPRoutePolicy %v: %v", hrpNN, err)
		}
		return f(httpRoutePolicy, httpRoutePolicy.Status), nil
	})
}

func addFileToZip(zipWriter *zip.Writer, fileName, fileContent string) error {
	file, err := zipWriter.Create(fileName)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte(fileContent))
	return err
}

func parentRefToString(p gatewayv1.ParentReference) string {
	if p.Namespace != nil && *p.Namespace != "" {
		return fmt.Sprintf("%v/%v", p.Namespace, p.Name)
	}
	return string(p.Name)
}

func findConditionInList(t testing.TestingT, conditions []metav1.Condition, expected metav1.Condition) bool {
	for _, cond := range conditions {
		if cond.Type == expected.Type {
			// an empty Status string means "Match any status".
			if expected.Status == "" || cond.Status == expected.Status {
				// an empty Reason string means "Match any reason".
				if expected.Reason == "" || cond.Reason == expected.Reason {
					return true
				}
				log.Printf("%s condition Reason set to %s, expected %s", expected.Type, cond.Reason, expected.Reason)
			}

			log.Printf("%s condition set to Status %s with Reason %v, expected Status %s", expected.Type, cond.Status, cond.Reason, expected.Status)
		}
	}

	log.Printf("%s was not in conditions list [%v]", expected.Type, conditions)
	return false
}
