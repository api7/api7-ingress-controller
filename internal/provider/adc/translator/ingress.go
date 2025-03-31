package translator

import (
	"fmt"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/internal/controller/label"
	"github.com/api7/api7-ingress-controller/internal/id"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (t *Translator) translateIngressTLS(ingressTLS *networkingv1.IngressTLS, secret *corev1.Secret, labels map[string]string) (*adctypes.SSL, error) {
	// extract the key pair from the secret
	cert, key, err := extractKeyPair(secret, true)
	if err != nil {
		return nil, err
	}

	hosts := ingressTLS.Hosts
	certHosts, err := extractHost(cert)
	if err != nil {
		return nil, err
	}
	hosts = append(hosts, certHosts...)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts found in ingress TLS")
	}

	ssl := &adctypes.SSL{
		Metadata: adctypes.Metadata{
			Labels: labels,
		},
		Certificates: []adctypes.Certificate{
			{
				Certificate: string(cert),
				Key:         string(key),
			},
		},
		Snis: hosts,
	}
	ssl.ID = id.GenID(string(cert))

	return ssl, nil
}

func (t *Translator) TranslateIngress(tctx *provider.TranslateContext, obj *networkingv1.Ingress) (*TranslateResult, error) {
	result := &TranslateResult{}

	labels := label.GenLabel(obj)

	// handle TLS configuration, convert to SSL objects
	for _, tls := range obj.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}
		secret := tctx.Secrets[types.NamespacedName{
			Namespace: obj.Namespace,
			Name:      tls.SecretName,
		}]
		if secret == nil {
			continue
		}
		if secret.Data == nil {
			log.Warnw("secret data is nil", zap.String("secret", secret.Namespace+"/"+secret.Name))
			continue
		}
		ssl, err := t.translateIngressTLS(&tls, secret, labels)
		if err != nil {
			return nil, err
		}

		result.SSL = append(result.SSL, ssl)
	}

	// process Ingress rules, convert to Service and Route objects
	for i, rule := range obj.Spec.Rules {
		// extract hostnames
		var hosts []string
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
		// if there is no HTTP path, skip
		if rule.HTTP == nil {
			continue
		}

		// create a service for each path
		for j, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}

			service := adctypes.NewDefaultService()
			service.Labels = labels
			service.Name = adctypes.ComposeServiceNameWithRule(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", i, j))
			service.ID = id.GenID(service.Name)
			service.Hosts = hosts

			// create an upstream
			upstream := adctypes.NewDefaultUpstream()

			// get the EndpointSlice of the backend service
			backendService := path.Backend.Service
			endpointSlices := tctx.EndpointSlices[types.NamespacedName{
				Namespace: obj.Namespace,
				Name:      backendService.Name,
			}]

			// get the service port configuration
			var servicePort int32 = 0
			var servicePortName string
			if backendService.Port.Number != 0 {
				servicePort = backendService.Port.Number
			} else if backendService.Port.Name != "" {
				servicePortName = backendService.Port.Name
			}

			// convert the EndpointSlice to upstream nodes
			if len(endpointSlices) > 0 {
				upstream.Nodes = t.translateEndpointSliceForIngress(1, endpointSlices, servicePort, servicePortName)
			}

			// if there is no upstream node, create a placeholder node
			if len(upstream.Nodes) == 0 {
				upstream.Nodes = adctypes.UpstreamNodes{
					{
						Host:   "0.0.0.0",
						Port:   int(servicePort),
						Weight: 1,
					},
				}
			}

			service.Upstream = upstream

			// create a route
			route := adctypes.NewDefaultRoute()
			route.Name = adctypes.ComposeRouteName(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", i, j))
			route.ID = id.GenID(route.Name)
			route.Labels = labels

			// set the path matching rule
			switch *path.PathType {
			case networkingv1.PathTypeExact:
				route.Uris = []string{path.Path}
			case networkingv1.PathTypePrefix:
				route.Uris = []string{path.Path + "*"}
			case networkingv1.PathTypeImplementationSpecific:
				route.Uris = []string{path.Path + "*"}
			}

			service.Routes = []*adctypes.Route{route}
			result.Services = append(result.Services, service)
		}
	}

	return result, nil
}

// translateEndpointSliceForIngress create upstream nodes from EndpointSlice
func (t *Translator) translateEndpointSliceForIngress(weight int, endpointSlices []discoveryv1.EndpointSlice, portNumber int32, portName string) adctypes.UpstreamNodes {
	var nodes adctypes.UpstreamNodes
	if len(endpointSlices) == 0 {
		return nodes
	}

	for _, endpointSlice := range endpointSlices {
		for _, port := range endpointSlice.Ports {
			// if the port number or port name is specified, only use the matching port
			if (portNumber != 0 && *port.Port != portNumber) || (portName != "" && *port.Name != portName) {
				continue
			}
			for _, endpoint := range endpointSlice.Endpoints {
				for _, addr := range endpoint.Addresses {
					node := adctypes.UpstreamNode{
						Host:   addr,
						Port:   int(*port.Port),
						Weight: weight,
					}
					nodes = append(nodes, node)
				}
			}
		}
	}

	return nodes
}
