// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"reflect"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestConvertIngressV1beta1ToV1(t *testing.T) {
	pathType := networkingv1beta1.PathTypeImplementationSpecific
	className := "apisix"

	beta := &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "ns",
		},
		Spec: networkingv1beta1.IngressSpec{
			IngressClassName: &className,
			Backend: &networkingv1beta1.IngressBackend{
				ServiceName: "svc",
				ServicePort: intstr.FromInt(80),
			},
			TLS: []networkingv1beta1.IngressTLS{
				{
					Hosts:      []string{"example.com"},
					SecretName: "secret",
				},
			},
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: "svc",
										ServicePort: intstr.FromString("http"),
									},
								},
							},
						},
					},
				},
			},
		},
		Status: networkingv1beta1.IngressStatus{
			LoadBalancer: networkingv1beta1.IngressLoadBalancerStatus{
				Ingress: []networkingv1beta1.IngressLoadBalancerIngress{
					{
						IP: "1.1.1.1",
						Ports: []networkingv1beta1.IngressPortStatus{
							{Port: 80},
						},
					},
				},
			},
		},
	}

	v1 := ConvertIngressV1beta1ToV1(beta)
	if v1 == nil {
		t.Fatalf("conversion returned nil")
	}

	if got := v1.Spec.DefaultBackend; got == nil || got.Service == nil || got.Service.Port.Number != 80 {
		t.Fatalf("default backend port not converted correctly: %#v", got)
	}
	if got := v1.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name; got != "http" {
		t.Fatalf("path backend service port name mismatch, got %q", got)
	}
	if v1.Spec.Rules[0].HTTP.Paths[0].PathType == nil || *v1.Spec.Rules[0].HTTP.Paths[0].PathType != networkingv1.PathTypeImplementationSpecific {
		t.Fatalf("path type not converted correctly")
	}
	if !reflect.DeepEqual(v1.Spec.TLS[0].Hosts, []string{"example.com"}) {
		t.Fatalf("tls hosts lost during conversion: %#v", v1.Spec.TLS[0].Hosts)
	}
	if v1.Status.LoadBalancer.Ingress[0].IP != "1.1.1.1" {
		t.Fatalf("status load balancer not converted, got %#v", v1.Status.LoadBalancer.Ingress)
	}
}

func TestConvertIngressV1ToV1beta1(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	className := "apisix"

	v1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "ns",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "svc",
					Port: networkingv1.ServiceBackendPort{Number: 8080},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"example.org"},
					SecretName: "secret",
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.org",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/api",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "svc",
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{
					{
						Hostname: "lb.example.org",
					},
				},
			},
		},
	}

	beta := ConvertIngressV1ToV1beta1(v1)
	if beta == nil {
		t.Fatalf("conversion returned nil")
	}

	if got := beta.Spec.Backend; got == nil || got.ServicePort.IntValue() != 8080 {
		t.Fatalf("default backend port not preserved: %#v", got)
	}
	if got := beta.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort.String(); got != "http" {
		t.Fatalf("path backend port name mismatch, got %q", got)
	}
	if beta.Spec.Rules[0].HTTP.Paths[0].PathType == nil || *beta.Spec.Rules[0].HTTP.Paths[0].PathType != networkingv1beta1.PathTypePrefix {
		t.Fatalf("path type not converted correctly")
	}
	if beta.Status.LoadBalancer.Ingress[0].Hostname != "lb.example.org" {
		t.Fatalf("status hostname not preserved")
	}
}
