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
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "IPv4 address",
			ip:   "192.168.1.1",
			want: false,
		},
		{
			name: "IPv6 address",
			ip:   "2001:db8::1",
			want: true,
		},
		{
			name: "IPv6 loopback",
			ip:   "::1",
			want: true,
		},
		{
			name: "IPv4 loopback",
			ip:   "127.0.0.1",
			want: false,
		},
		{
			name: "invalid IP",
			ip:   "invalid",
			want: false,
		},
		{
			name: "IPv4-mapped IPv6",
			ip:   "::ffff:192.168.1.1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIPv6(tt.ip); got != tt.want {
				t.Errorf("isIPv6() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertEndpointsToEndpointSlice(t *testing.T) {
	tests := []struct {
		name      string
		endpoints *corev1.Endpoints
		want      []discoveryv1.EndpointSlice
	}{
		{
			name:      "nil endpoints",
			endpoints: nil,
			want:      nil,
		},
		{
			name: "empty subsets",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports:       []discoveryv1.EndpointPort{},
					Endpoints:   []discoveryv1.Endpoint{},
				},
			},
		},
		{
			name: "IPv4 only",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
							{
								IP: "192.168.1.2",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
								Name:     "http",
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"192.168.1.1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
						{
							Addresses: []string{"192.168.1.2"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name: "IPv6 only",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "2001:db8::1",
							},
							{
								IP: "2001:db8::2",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
								Name:     "http",
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v6",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv6,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"2001:db8::1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
						{
							Addresses: []string{"2001:db8::2"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name: "dual stack (IPv4 and IPv6)",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
							{
								IP: "2001:db8::1",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
								Name:     "http",
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"192.168.1.1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v6",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv6,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"2001:db8::1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name: "ready and not ready addresses",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
						},
						NotReadyAddresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.2",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
								Name:     "http",
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"192.168.1.1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
						{
							Addresses: []string{"192.168.1.2"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(false),
							},
						},
					},
				},
			},
		},
		{
			name: "multiple subsets",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
								Name:     "http",
							},
						},
					},
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "2001:db8::1",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     443,
								Protocol: corev1.ProtocolTCP,
								Name:     "https",
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("http"),
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"192.168.1.1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-1-v6",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv6,
					Ports: []discoveryv1.EndpointPort{
						{
							Name:     ptr.To("https"),
							Port:     ptr.To[int32](443),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"2001:db8::1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name: "with target ref and hostname",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP:       "192.168.1.1",
								Hostname: "pod-1",
								TargetRef: &corev1.ObjectReference{
									Kind:      "Pod",
									Name:      "pod-1",
									Namespace: "default",
								},
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
							},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-0-v4",
						Namespace: "default",
						Labels: map[string]string{
							discoveryv1.LabelServiceName: "test-service",
						},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
					Ports: []discoveryv1.EndpointPort{
						{
							Port:     ptr.To[int32](80),
							Protocol: ptr.To(corev1.ProtocolTCP),
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"192.168.1.1"},
							Conditions: discoveryv1.EndpointConditions{
								Ready: ptr.To(true),
							},
							Hostname: ptr.To("pod-1"),
							TargetRef: &corev1.ObjectReference{
								Kind:      "Pod",
								Name:      "pod-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEndpointsToEndpointSlice(tt.endpoints)
			if len(got) != len(tt.want) {
				t.Errorf("ConvertEndpointsToEndpointSlice() returned %d slices, want %d", len(got), len(tt.want))
				return
			}

			for i, slice := range got {
				if i >= len(tt.want) {
					t.Errorf("ConvertEndpointsToEndpointSlice() returned more slices than expected")
					return
				}

				wantSlice := tt.want[i]

				// Check ObjectMeta
				if slice.Name != wantSlice.Name {
					t.Errorf("EndpointSlice[%d].Name = %v, want %v", i, slice.Name, wantSlice.Name)
				}
				if slice.Namespace != wantSlice.Namespace {
					t.Errorf("EndpointSlice[%d].Namespace = %v, want %v", i, slice.Namespace, wantSlice.Namespace)
				}
				if len(slice.Labels) != len(wantSlice.Labels) {
					t.Errorf("EndpointSlice[%d].Labels length = %v, want %v", i, len(slice.Labels), len(wantSlice.Labels))
				}
				for k, v := range wantSlice.Labels {
					if slice.Labels[k] != v {
						t.Errorf("EndpointSlice[%d].Labels[%s] = %v, want %v", i, k, slice.Labels[k], v)
					}
				}

				// Check AddressType
				if slice.AddressType != wantSlice.AddressType {
					t.Errorf("EndpointSlice[%d].AddressType = %v, want %v", i, slice.AddressType, wantSlice.AddressType)
				}

				// Check Ports
				if len(slice.Ports) != len(wantSlice.Ports) {
					t.Errorf("EndpointSlice[%d].Ports length = %v, want %v", i, len(slice.Ports), len(wantSlice.Ports))
				}
				for j, port := range slice.Ports {
					if j >= len(wantSlice.Ports) {
						continue
					}
					wantPort := wantSlice.Ports[j]
					if (port.Name == nil) != (wantPort.Name == nil) || (port.Name != nil && *port.Name != *wantPort.Name) {
						t.Errorf("EndpointSlice[%d].Ports[%d].Name = %v, want %v", i, j, port.Name, wantPort.Name)
					}
					if *port.Port != *wantPort.Port {
						t.Errorf("EndpointSlice[%d].Ports[%d].Port = %v, want %v", i, j, *port.Port, *wantPort.Port)
					}
					if *port.Protocol != *wantPort.Protocol {
						t.Errorf("EndpointSlice[%d].Ports[%d].Protocol = %v, want %v", i, j, *port.Protocol, *wantPort.Protocol)
					}
				}

				// Check Endpoints
				if len(slice.Endpoints) != len(wantSlice.Endpoints) {
					t.Errorf("EndpointSlice[%d].Endpoints length = %v, want %v", i, len(slice.Endpoints), len(wantSlice.Endpoints))
				}
				for j, endpoint := range slice.Endpoints {
					if j >= len(wantSlice.Endpoints) {
						continue
					}
					wantEndpoint := wantSlice.Endpoints[j]
					if len(endpoint.Addresses) != len(wantEndpoint.Addresses) {
						t.Errorf("EndpointSlice[%d].Endpoints[%d].Addresses length = %v, want %v", i, j, len(endpoint.Addresses), len(wantEndpoint.Addresses))
					}
					for k, addr := range endpoint.Addresses {
						if k >= len(wantEndpoint.Addresses) {
							continue
						}
						if addr != wantEndpoint.Addresses[k] {
							t.Errorf("EndpointSlice[%d].Endpoints[%d].Addresses[%d] = %v, want %v", i, j, k, addr, wantEndpoint.Addresses[k])
						}
					}
					if *endpoint.Conditions.Ready != *wantEndpoint.Conditions.Ready {
						t.Errorf("EndpointSlice[%d].Endpoints[%d].Conditions.Ready = %v, want %v", i, j, *endpoint.Conditions.Ready, *wantEndpoint.Conditions.Ready)
					}
					if (endpoint.Hostname == nil) != (wantEndpoint.Hostname == nil) || (endpoint.Hostname != nil && *endpoint.Hostname != *wantEndpoint.Hostname) {
						t.Errorf("EndpointSlice[%d].Endpoints[%d].Hostname = %v, want %v", i, j, endpoint.Hostname, wantEndpoint.Hostname)
					}
				}
			}
		})
	}
}
