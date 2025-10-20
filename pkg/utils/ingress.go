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
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ConvertIngressToV1 returns a v1 Ingress representation regardless of the source version.
func ConvertIngressToV1(obj any, scheme *runtime.Scheme) (*networkingv1.Ingress, error) {
	switch ingress := obj.(type) {
	case *networkingv1.Ingress:
		return ingress, nil
	case *networkingv1beta1.Ingress:
		out := new(networkingv1.Ingress)
		if err := scheme.Convert(ingress, out, nil); err != nil {
			return nil, fmt.Errorf("convert ingress v1beta1 to v1: %w", err)
		}
		out.APIVersion = networkingv1.SchemeGroupVersion.String()
		out.Kind = "Ingress"
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported ingress type %T", obj)
	}
}

// ConvertIngressFromV1 converts a v1 Ingress object into the target group version.
func ConvertIngressFromV1(ingress *networkingv1.Ingress, scheme *runtime.Scheme, gv schema.GroupVersion) (any, error) {
	switch gv {
	case networkingv1.SchemeGroupVersion:
		return ingress, nil
	case networkingv1beta1.SchemeGroupVersion:
		out := new(networkingv1beta1.Ingress)
		if err := scheme.Convert(ingress, out, nil); err != nil {
			return nil, fmt.Errorf("convert ingress v1 to v1beta1: %w", err)
		}
		out.APIVersion = networkingv1beta1.SchemeGroupVersion.String()
		out.Kind = "Ingress"
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported ingress group version: %s", gv.String())
	}
}
