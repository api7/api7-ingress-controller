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
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ConvertEndpointsToEndpointSlice converts a Kubernetes Endpoints object to an EndpointSlice object.
// This function is used to provide backward compatibility for Kubernetes 1.18 clusters that don't
// have EndpointSlice support but still use the older Endpoints API.
//
// The conversion follows these rules:
// - Each Endpoints subset becomes a separate EndpointSlice
// - Endpoint addresses are mapped to EndpointSlice endpoints
// - Port information is preserved
// - Ready state is mapped from Endpoints addresses vs notReadyAddresses
//
// Note: Some EndpointSlice features like topology and conditions may not be fully represented
// since they don't exist in the Endpoints API.
func ConvertEndpointsToEndpointSlice(ep *corev1.Endpoints) []discoveryv1.EndpointSlice {
	if ep == nil {
		return nil
	}

	//nolint:prealloc
	var endpointSlices []discoveryv1.EndpointSlice

	// If there are no subsets, create an empty EndpointSlice
	if len(ep.Subsets) == 0 {
		endpointSlice := discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ep.Name,
				Namespace: ep.Namespace,
				Labels: map[string]string{
					discoveryv1.LabelServiceName: ep.Name,
				},
			},
			AddressType: discoveryv1.AddressTypeIPv4, // Default to IPv4
			Ports:       []discoveryv1.EndpointPort{},
			Endpoints:   []discoveryv1.Endpoint{},
		}
		endpointSlices = append(endpointSlices, endpointSlice)
		return endpointSlices
	}

	// Convert each subset to an EndpointSlice
	for i, subset := range ep.Subsets {
		// Create ports array
		var ports []discoveryv1.EndpointPort
		for _, port := range subset.Ports {
			endpointPort := discoveryv1.EndpointPort{
				Name:     &port.Name,
				Port:     &port.Port,
				Protocol: &port.Protocol,
			}
			ports = append(ports, endpointPort)
		}

		// Create endpoints array from addresses (ready endpoints)
		var endpoints []discoveryv1.Endpoint
		for _, addr := range subset.Addresses {
			endpoint := discoveryv1.Endpoint{
				Addresses: []string{addr.IP},
				Conditions: discoveryv1.EndpointConditions{
					Ready: ptr.To(true), // Addresses in Endpoints.Addresses are ready
				},
			}

			// Add target ref if available
			if addr.TargetRef != nil {
				endpoint.TargetRef = addr.TargetRef
			}

			// Add hostname if available
			if addr.Hostname != "" {
				endpoint.Hostname = &addr.Hostname
			}

			endpoints = append(endpoints, endpoint)
		}

		// Add not ready addresses
		for _, addr := range subset.NotReadyAddresses {
			endpoint := discoveryv1.Endpoint{
				Addresses: []string{addr.IP},
				Conditions: discoveryv1.EndpointConditions{
					Ready: ptr.To(false), // NotReadyAddresses are not ready
				},
			}

			// Add target ref if available
			if addr.TargetRef != nil {
				endpoint.TargetRef = addr.TargetRef
			}

			// Add hostname if available
			if addr.Hostname != "" {
				endpoint.Hostname = &addr.Hostname
			}

			endpoints = append(endpoints, endpoint)
		}

		// Determine address type based on first endpoint
		addressType := discoveryv1.AddressTypeIPv4
		if len(endpoints) > 0 && len(endpoints[0].Addresses) > 0 {
			// Simple IPv6 detection - if address contains colons, assume IPv6
			if containsColon(endpoints[0].Addresses[0]) {
				addressType = discoveryv1.AddressTypeIPv6
			}
		}

		// Create EndpointSlice name with suffix if multiple subsets
		name := ep.Name
		if len(ep.Subsets) > 1 {
			name = ep.Name + "-" + string(rune('a'+i))
		}

		endpointSlice := discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ep.Namespace,
				Labels: map[string]string{
					discoveryv1.LabelServiceName: ep.Name,
				},
				// Copy owner references if they exist
				OwnerReferences: ep.OwnerReferences,
			},
			AddressType: addressType,
			Ports:       ports,
			Endpoints:   endpoints,
		}

		endpointSlices = append(endpointSlices, endpointSlice)
	}

	return endpointSlices
}

// containsColon is a simple helper to detect IPv6 addresses
func containsColon(addr string) bool {
	for _, char := range addr {
		if char == ':' {
			return true
		}
	}
	return false
}
