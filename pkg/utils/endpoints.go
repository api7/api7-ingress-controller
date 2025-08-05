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
	"net"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ConvertEndpointsToEndpointSlice converts a Kubernetes Endpoints object to one
// or more EndpointSlice objects, supporting IPv4/IPv6 dual stack.
// This function is used to provide backward compatibility for Kubernetes 1.18 clusters that don't
// have EndpointSlice support but still use the older Endpoints API.
//
// The conversion follows these rules:
// - Each Endpoints subset is split into separate IPv4 and IPv6 EndpointSlices
// - Uses net.ParseIP for reliable address family detection instead of string matching
// - IPv4 and IPv6 endpoints from the same subset are separated into different slices
// - Naming convention: <svc-name>-<subset-index>-v4 / -v6
// - Port information is preserved
// - Ready state is mapped from Endpoints addresses vs notReadyAddresses
//
// Note: Some EndpointSlice features like topology and conditions may not be fully represented
// since they don't exist in the Endpoints API.
func ConvertEndpointsToEndpointSlice(ep *corev1.Endpoints) []discoveryv1.EndpointSlice {
	if ep == nil {
		return nil
	}

	var endpointSlices []discoveryv1.EndpointSlice

	// If there are no subsets, create an empty EndpointSlice
	if len(ep.Subsets) == 0 {
		endpointSlices = append(endpointSlices, discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:            ep.Name + "-v4", // Default to v4
				Namespace:       ep.Namespace,
				Labels:          map[string]string{discoveryv1.LabelServiceName: ep.Name},
				OwnerReferences: ep.OwnerReferences,
			},
			AddressType: discoveryv1.AddressTypeIPv4,
			Ports:       []discoveryv1.EndpointPort{},
			Endpoints:   []discoveryv1.Endpoint{},
		})
		return endpointSlices
	}

	for i, subset := range ep.Subsets {
		// Create ports array
		ports := make([]discoveryv1.EndpointPort, 0, len(subset.Ports))
		for _, p := range subset.Ports {
			epPort := discoveryv1.EndpointPort{
				Port:     &p.Port,
				Protocol: &p.Protocol,
				Name:     &p.Name,
			}
			ports = append(ports, epPort)
		}

		// Separate IPv4 and IPv6 addresses
		var (
			ipv4Endpoints []discoveryv1.Endpoint
			ipv6Endpoints []discoveryv1.Endpoint
		)
		buildEndpoint := func(addr corev1.EndpointAddress, ready bool) discoveryv1.Endpoint {
			e := discoveryv1.Endpoint{
				Addresses: []string{addr.IP},
				Conditions: discoveryv1.EndpointConditions{
					Ready: ptr.To(ready),
				},
			}
			if addr.TargetRef != nil {
				e.TargetRef = addr.TargetRef
			}
			if addr.Hostname != "" {
				e.Hostname = &addr.Hostname
			}
			return e
		}

		// Process ready addresses
		for _, a := range subset.Addresses {
			if isIPv6(a.IP) {
				ipv6Endpoints = append(ipv6Endpoints, buildEndpoint(a, true))
			} else {
				ipv4Endpoints = append(ipv4Endpoints, buildEndpoint(a, true))
			}
		}
		// Process not ready addresses
		for _, a := range subset.NotReadyAddresses {
			if isIPv6(a.IP) {
				ipv6Endpoints = append(ipv6Endpoints, buildEndpoint(a, false))
			} else {
				ipv4Endpoints = append(ipv4Endpoints, buildEndpoint(a, false))
			}
		}

		// Create EndpointSlices for each address type
		makeSlice := func(suffix string, addrType discoveryv1.AddressType, eps []discoveryv1.Endpoint) discoveryv1.EndpointSlice {
			return discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:            fmt.Sprintf("%s-%d-%s", ep.Name, i, suffix),
					Namespace:       ep.Namespace,
					Labels:          map[string]string{discoveryv1.LabelServiceName: ep.Name},
					OwnerReferences: ep.OwnerReferences,
				},
				AddressType: addrType,
				Ports:       ports,
				Endpoints:   eps,
			}
		}

		if len(ipv4Endpoints) > 0 {
			endpointSlices = append(endpointSlices, makeSlice("v4", discoveryv1.AddressTypeIPv4, ipv4Endpoints))
		}
		if len(ipv6Endpoints) > 0 {
			endpointSlices = append(endpointSlices, makeSlice("v6", discoveryv1.AddressTypeIPv6, ipv6Endpoints))
		}
	}

	return endpointSlices
}

// isIPv6 uses net.ParseIP to determine if an IP address is IPv6
func isIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() == nil
}
