package scaffold

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	_locustConfigMapTemplate = `
apiVersion: v1 
kind: ConfigMap 
metadata: 
  name: locust-config 
data: 
  locustfile.py: |- 
    from locust import HttpUser, task, between 

    class HttpbinRequester(HttpUser):
        @task 
        def headers(self): 
            self.client.get("/headers", headers={"Host": "httpbin.example"})

        @task
        def get(self):
            self.client.get("/get", headers={"Host": "httpbin.example"})

        @task
        def post(self):
            self.client.post("/post", headers={"Host": "httpbin.example"})

        @task
        def image(self):
            self.client.image("/image", headers={"Host": "httpbin.example"})

  LOCUST_HOST: http://api7ee3-apisix-gateway-mtls:9080
  LOCUST_SPAWN_RATE: "50"
  LOCUST_USERS: "500"
  LOCUST_AUTOSTART: "true"
`
	_locustDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: locust
spec:
  selector:
    matchLabels:
      app: locust
  template:
    metadata:
      labels:
        app: locust
    spec:
      containers:
        - name: locust
          image: locustio/locust
          ports:
            - containerPort: 8089
          env:
            - name: LOCUST_HOST
              valueFrom:
                configMapKeyRef:
                  name: locust-config
                  key: LOCUST_HOST
            - name: LOCUST_SPAWN_RATE
              valueFrom:
                configMapKeyRef:
                  name: locust-config
                  key: LOCUST_SPAWN_RATE
            - name: LOCUST_USERS
              valueFrom:
                configMapKeyRef:
                  name: locust-config
                  key: LOCUST_USERS
            - name: LOCUST_AUTOSTART
              valueFrom:
                configMapKeyRef:
                  name: locust-config
                  key: LOCUST_AUTOSTART
          volumeMounts:
            - mountPath: /home/locust
              name: locust-config
      volumes:
        - name: locust-config
          configMap:
            name: locust-config
`
	_locustServiceTemplate = `
apiVersion: v1 
kind: Service 
metadata: 
  name: locust 
spec: 
  selector: 
    app: locust 
  ports: 
    - name: web
      port: 8089 
      targetPort: 8089 
      protocol: TCP
  type: ClusterIP `
)

func (s *Scaffold) DeployLocust() *corev1.Service {
	// create ConfigMap, Deployment, Service
	for _, yaml_ := range []string{_locustConfigMapTemplate, _locustDeploymentTemplate, _locustServiceTemplate} {
		err := s.CreateResourceFromString(yaml_)
		Expect(err).NotTo(HaveOccurred(), "create resource: %s", yaml_)
	}

	service, err := k8s.GetServiceE(s.t, s.kubectlOptions, "locust")
	Expect(err).NotTo(HaveOccurred(), "get service: locust")

	s.EnsureNumEndpointsReady(s.t, service.Name, 1)
	s.locustTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "locust", 8089, 8089)
	s.addFinalizers(s.locustTunnel.Close)

	err = s.locustTunnel.ForwardPortE(s.t)
	Expect(err).NotTo(HaveOccurred(), "port-forward service: locust")

	return service
}

func (s *Scaffold) ResetLocust() error {
	if s.locustTunnel == nil {
		return errors.New("locust is not deployed")
	}
	resp, err := http.Get("http://" + s.locustTunnel.Endpoint() + "/stats/reset")
	if err != nil {
		return errors.Wrap(err, "failed to request reset locust")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("request reset locust not OK, status: %s", resp.Status)
	}
	return nil
}

func (s *Scaffold) DownloadLocustReport(filename string) error {
	if s.locustTunnel == nil {
		return errors.New("locust is not deployed")
	}
	if !strings.EqualFold(filepath.Ext(filename), ".html") {
		filename += ".html"
	}
	_ = os.Remove(filename)
	resp, err := http.Get("http://" + s.locustTunnel.Endpoint() + "/stats/report?download=1&theme=light")
	if err != nil {
		return errors.Wrap(err, "failed to request download report")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("request download report not OK, status: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read report")
	}
	return os.WriteFile(filename, data, 0644)
}
