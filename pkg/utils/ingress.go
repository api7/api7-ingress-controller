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
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ConvertIngressV1beta1ToV1 transforms a networking.k8s.io/v1beta1 Ingress into its v1 counterpart.
// This allows callers to reuse the same processing pipeline that already targets the v1 API.
func ConvertIngressV1beta1ToV1(src *networkingv1beta1.Ingress) *networkingv1.Ingress {
	if src == nil {
		return nil
	}

	out := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: *src.ObjectMeta.DeepCopy(),
		Status:     ConvertIngressStatusV1beta1ToV1(src.Status),
	}
	out.Spec = convertIngressSpecV1beta1ToV1(&src.Spec)

	return out
}

// ConvertIngressV1ToV1beta1 maps a v1 Ingress back to the legacy v1beta1 shape.
// Primarily used for status updates in legacy clusters.
func ConvertIngressV1ToV1beta1(src *networkingv1.Ingress) *networkingv1beta1.Ingress {
	if src == nil {
		return nil
	}

	out := &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkingv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: *src.ObjectMeta.DeepCopy(),
		Status:     ConvertIngressStatusV1ToV1beta1(src.Status),
	}
	out.Spec = convertIngressSpecV1ToV1beta1(&src.Spec)

	return out
}

func convertIngressSpecV1beta1ToV1(src *networkingv1beta1.IngressSpec) networkingv1.IngressSpec {
	if src == nil {
		return networkingv1.IngressSpec{}
	}

	out := networkingv1.IngressSpec{}
	if src.IngressClassName != nil {
		out.IngressClassName = src.IngressClassName
	}
	if len(src.TLS) > 0 {
		out.TLS = convertIngressTLSV1beta1ToV1(src.TLS)
	}
	if src.Backend != nil {
		out.DefaultBackend = convertIngressBackendV1beta1ToV1(src.Backend)
	}
	if len(src.Rules) > 0 {
		out.Rules = convertIngressRulesV1beta1ToV1(src.Rules)
	}

	return out
}

func convertIngressSpecV1ToV1beta1(src *networkingv1.IngressSpec) networkingv1beta1.IngressSpec {
	if src == nil {
		return networkingv1beta1.IngressSpec{}
	}

	out := networkingv1beta1.IngressSpec{}
	if src.IngressClassName != nil {
		out.IngressClassName = src.IngressClassName
	}
	if len(src.TLS) > 0 {
		out.TLS = convertIngressTLSV1ToV1beta1(src.TLS)
	}
	if src.DefaultBackend != nil {
		out.Backend = convertIngressBackendV1ToV1beta1(src.DefaultBackend)
	}
	if len(src.Rules) > 0 {
		out.Rules = convertIngressRulesV1ToV1beta1(src.Rules)
	}

	return out
}

func convertIngressTLSV1beta1ToV1(src []networkingv1beta1.IngressTLS) []networkingv1.IngressTLS {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1.IngressTLS, len(src))
	for i := range src {
		t := src[i]
		out[i] = networkingv1.IngressTLS{
			Hosts:      append([]string(nil), t.Hosts...),
			SecretName: t.SecretName,
		}
	}

	return out
}

func convertIngressTLSV1ToV1beta1(src []networkingv1.IngressTLS) []networkingv1beta1.IngressTLS {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1beta1.IngressTLS, len(src))
	for i := range src {
		t := src[i]
		out[i] = networkingv1beta1.IngressTLS{
			Hosts:      append([]string(nil), t.Hosts...),
			SecretName: t.SecretName,
		}
	}

	return out
}

func convertIngressRulesV1beta1ToV1(src []networkingv1beta1.IngressRule) []networkingv1.IngressRule {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1.IngressRule, len(src))
	for i := range src {
		rule := src[i]
		out[i].Host = rule.Host
		if rule.HTTP != nil {
			out[i].HTTP = &networkingv1.HTTPIngressRuleValue{
				Paths: convertHTTPIngressPathsV1beta1ToV1(rule.HTTP.Paths),
			}
		}
	}

	return out
}

func convertIngressRulesV1ToV1beta1(src []networkingv1.IngressRule) []networkingv1beta1.IngressRule {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1beta1.IngressRule, len(src))
	for i := range src {
		rule := src[i]
		out[i].Host = rule.Host
		if rule.HTTP != nil {
			out[i].HTTP = &networkingv1beta1.HTTPIngressRuleValue{
				Paths: convertHTTPIngressPathsV1ToV1beta1(rule.HTTP.Paths),
			}
		}
	}

	return out
}

func convertHTTPIngressPathsV1beta1ToV1(src []networkingv1beta1.HTTPIngressPath) []networkingv1.HTTPIngressPath {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1.HTTPIngressPath, len(src))
	for i := range src {
		path := src[i]
		out[i] = networkingv1.HTTPIngressPath{
			Path:     path.Path,
			PathType: convertPathTypeV1beta1ToV1(path.PathType),
		}
		if backend := convertIngressBackendV1beta1ToV1(&path.Backend); backend != nil {
			out[i].Backend = *backend
		}
	}

	return out
}

func convertHTTPIngressPathsV1ToV1beta1(src []networkingv1.HTTPIngressPath) []networkingv1beta1.HTTPIngressPath {
	if len(src) == 0 {
		return nil
	}

	out := make([]networkingv1beta1.HTTPIngressPath, len(src))
	for i := range src {
		path := src[i]
		out[i] = networkingv1beta1.HTTPIngressPath{
			Path:     path.Path,
			PathType: convertPathTypeV1ToV1beta1(path.PathType),
		}
		if backend := convertIngressBackendV1ToV1beta1(&path.Backend); backend != nil {
			out[i].Backend = *backend
		}
	}

	return out
}

func convertIngressBackendV1beta1ToV1(backend *networkingv1beta1.IngressBackend) *networkingv1.IngressBackend {
	if backend == nil {
		return nil
	}

	out := &networkingv1.IngressBackend{
		Resource: backend.Resource,
	}

	service := &networkingv1.IngressServiceBackend{}
	hasService := false

	if backend.ServiceName != "" {
		service.Name = backend.ServiceName
		hasService = true
	}

	switch backend.ServicePort.Type {
	case intstr.Int:
		service.Port.Number = int32(backend.ServicePort.IntValue())
		hasService = true
	case intstr.String:
		service.Port.Name = backend.ServicePort.String()
		hasService = true
	}

	if hasService {
		out.Service = service
	}

	return out
}

func convertIngressBackendV1ToV1beta1(backend *networkingv1.IngressBackend) *networkingv1beta1.IngressBackend {
	if backend == nil {
		return nil
	}

	out := &networkingv1beta1.IngressBackend{
		Resource: backend.Resource,
	}

	if backend.Service != nil {
		out.ServiceName = backend.Service.Name
		switch {
		case backend.Service.Port.Number != 0:
			out.ServicePort = intstr.FromInt(int(backend.Service.Port.Number))
		case backend.Service.Port.Name != "":
			out.ServicePort = intstr.FromString(backend.Service.Port.Name)
		}
	}

	return out
}

func convertPathTypeV1beta1ToV1(pathType *networkingv1beta1.PathType) *networkingv1.PathType {
	if pathType == nil {
		return nil
	}
	pt := networkingv1.PathType(*pathType)
	return &pt
}

func convertPathTypeV1ToV1beta1(pathType *networkingv1.PathType) *networkingv1beta1.PathType {
	if pathType == nil {
		return nil
	}
	pt := networkingv1beta1.PathType(*pathType)
	return &pt
}

func ConvertIngressStatusV1beta1ToV1(status networkingv1beta1.IngressStatus) networkingv1.IngressStatus {
	out := networkingv1.IngressStatus{}
	if len(status.LoadBalancer.Ingress) > 0 {
		out.LoadBalancer.Ingress = make([]networkingv1.IngressLoadBalancerIngress, len(status.LoadBalancer.Ingress))
		for i := range status.LoadBalancer.Ingress {
			ing := status.LoadBalancer.Ingress[i]
			out.LoadBalancer.Ingress[i] = networkingv1.IngressLoadBalancerIngress{
				IP:       ing.IP,
				Hostname: ing.Hostname,
				Ports:    convertLoadBalancerPortsV1beta1ToV1(ing.Ports),
			}
		}
	}
	return out
}

func ConvertIngressStatusV1ToV1beta1(status networkingv1.IngressStatus) networkingv1beta1.IngressStatus {
	out := networkingv1beta1.IngressStatus{}
	if len(status.LoadBalancer.Ingress) > 0 {
		out.LoadBalancer.Ingress = make([]networkingv1beta1.IngressLoadBalancerIngress, len(status.LoadBalancer.Ingress))
		for i := range status.LoadBalancer.Ingress {
			ing := status.LoadBalancer.Ingress[i]
			out.LoadBalancer.Ingress[i] = networkingv1beta1.IngressLoadBalancerIngress{
				IP:       ing.IP,
				Hostname: ing.Hostname,
				Ports:    convertLoadBalancerPortsV1ToV1beta1(ing.Ports),
			}
		}
	}
	return out
}

func convertLoadBalancerPortsV1beta1ToV1(ports []networkingv1beta1.IngressPortStatus) []networkingv1.IngressPortStatus {
	if len(ports) == 0 {
		return nil
	}
	out := make([]networkingv1.IngressPortStatus, len(ports))
	for i := range ports {
		out[i] = networkingv1.IngressPortStatus{
			Port:     ports[i].Port,
			Protocol: ports[i].Protocol,
		}
	}
	return out
}

func convertLoadBalancerPortsV1ToV1beta1(ports []networkingv1.IngressPortStatus) []networkingv1beta1.IngressPortStatus {
	if len(ports) == 0 {
		return nil
	}
	out := make([]networkingv1beta1.IngressPortStatus, len(ports))
	for i := range ports {
		out[i] = networkingv1beta1.IngressPortStatus{
			Port:     ports[i].Port,
			Protocol: ports[i].Protocol,
		}
	}
	return out
}
