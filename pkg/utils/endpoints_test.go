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
							{IP: "192.168.1.1"},
							{IP: "192.168.1.2"},
						},
						Ports: []corev1.EndpointPort{
							{Port: 80, Protocol: corev1.ProtocolTCP, Name: "http"},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				createTestEndpointSlice("test-service-0-v4", discoveryv1.AddressTypeIPv4,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{
						createReadyEndpoint("192.168.1.1"),
						createReadyEndpoint("192.168.1.2"),
					}),
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
							{IP: "2001:db8::1"},
							{IP: "2001:db8::2"},
						},
						Ports: []corev1.EndpointPort{
							{Port: 80, Protocol: corev1.ProtocolTCP, Name: "http"},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				createTestEndpointSlice("test-service-0-v6", discoveryv1.AddressTypeIPv6,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{
						createReadyEndpoint("2001:db8::1"),
						createReadyEndpoint("2001:db8::2"),
					}),
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
							{IP: "192.168.1.1"},
							{IP: "2001:db8::1"},
						},
						Ports: []corev1.EndpointPort{
							{Port: 80, Protocol: corev1.ProtocolTCP, Name: "http"},
						},
					},
				},
			},
			want: []discoveryv1.EndpointSlice{
				createTestEndpointSlice("test-service-0-v4", discoveryv1.AddressTypeIPv4,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{createReadyEndpoint("192.168.1.1")}),
				createTestEndpointSlice("test-service-0-v6", discoveryv1.AddressTypeIPv6,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{createReadyEndpoint("2001:db8::1")}),
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
				createTestEndpointSlice("test-service-0-v4", discoveryv1.AddressTypeIPv4,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{
						createReadyEndpoint("192.168.1.1"),
						createNotReadyEndpoint("192.168.1.2"),
					}),
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
				createTestEndpointSlice("test-service-0-v4", discoveryv1.AddressTypeIPv4,
					[]discoveryv1.EndpointPort{createHTTPPort()},
					[]discoveryv1.Endpoint{createReadyEndpoint("192.168.1.1")}),
				createTestEndpointSlice("test-service-1-v6", discoveryv1.AddressTypeIPv6,
					[]discoveryv1.EndpointPort{createHTTPSPort(443)},
					[]discoveryv1.Endpoint{createReadyEndpoint("2001:db8::1")}),
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
				createTestEndpointSlice("test-service-0-v4", discoveryv1.AddressTypeIPv4,
					[]discoveryv1.EndpointPort{createPlainPort(80)},
					[]discoveryv1.Endpoint{createEndpointWithHostname("192.168.1.1", "pod-1")}),
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
				assertEndpointSliceEqual(t, slice, tt.want[i], i)
			}
		})
	}
}

func assertEndpointSliceEqual(t *testing.T, got, want discoveryv1.EndpointSlice, index int) {
	t.Helper()

	if got.Name != want.Name {
		t.Errorf("EndpointSlice[%d].Name = %v, want %v", index, got.Name, want.Name)
	}
	if got.Namespace != want.Namespace {
		t.Errorf("EndpointSlice[%d].Namespace = %v, want %v", index, got.Namespace, want.Namespace)
	}
	if len(got.Labels) != len(want.Labels) {
		t.Errorf("EndpointSlice[%d].Labels length = %v, want %v", index, len(got.Labels), len(want.Labels))
	}
	for k, v := range want.Labels {
		if got.Labels[k] != v {
			t.Errorf("EndpointSlice[%d].Labels[%s] = %v, want %v", index, k, got.Labels[k], v)
		}
	}

	if got.AddressType != want.AddressType {
		t.Errorf("EndpointSlice[%d].AddressType = %v, want %v", index, got.AddressType, want.AddressType)
	}

	assertEndpointPortsEqual(t, got.Ports, want.Ports, index)
	assertEndpointsEqual(t, got.Endpoints, want.Endpoints, index)
}

func assertEndpointPortsEqual(t *testing.T, got, want []discoveryv1.EndpointPort, sliceIndex int) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("EndpointSlice[%d].Ports length = %v, want %v", sliceIndex, len(got), len(want))
		return
	}

	for j, port := range got {
		if j >= len(want) {
			continue
		}
		wantPort := want[j]
		if (port.Name == nil) != (wantPort.Name == nil) || (port.Name != nil && *port.Name != *wantPort.Name) {
			t.Errorf("EndpointSlice[%d].Ports[%d].Name = %v, want %v", sliceIndex, j, port.Name, wantPort.Name)
		}
		if *port.Port != *wantPort.Port {
			t.Errorf("EndpointSlice[%d].Ports[%d].Port = %v, want %v", sliceIndex, j, *port.Port, *wantPort.Port)
		}
		if *port.Protocol != *wantPort.Protocol {
			t.Errorf("EndpointSlice[%d].Ports[%d].Protocol = %v, want %v", sliceIndex, j, *port.Protocol, *wantPort.Protocol)
		}
	}
}

func assertEndpointsEqual(t *testing.T, got, want []discoveryv1.Endpoint, sliceIndex int) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("EndpointSlice[%d].Endpoints length = %v, want %v", sliceIndex, len(got), len(want))
		return
	}

	for j, endpoint := range got {
		if j >= len(want) {
			continue
		}
		wantEndpoint := want[j]

		if len(endpoint.Addresses) != len(wantEndpoint.Addresses) {
			t.Errorf("EndpointSlice[%d].Endpoints[%d].Addresses length = %v, want %v", sliceIndex, j, len(endpoint.Addresses), len(wantEndpoint.Addresses))
		}
		for k, addr := range endpoint.Addresses {
			if k >= len(wantEndpoint.Addresses) {
				continue
			}
			if addr != wantEndpoint.Addresses[k] {
				t.Errorf("EndpointSlice[%d].Endpoints[%d].Addresses[%d] = %v, want %v", sliceIndex, j, k, addr, wantEndpoint.Addresses[k])
			}
		}

		if *endpoint.Conditions.Ready != *wantEndpoint.Conditions.Ready {
			t.Errorf("EndpointSlice[%d].Endpoints[%d].Conditions.Ready = %v, want %v", sliceIndex, j, *endpoint.Conditions.Ready, *wantEndpoint.Conditions.Ready)
		}

		if (endpoint.Hostname == nil) != (wantEndpoint.Hostname == nil) || (endpoint.Hostname != nil && *endpoint.Hostname != *wantEndpoint.Hostname) {
			t.Errorf("EndpointSlice[%d].Endpoints[%d].Hostname = %v, want %v", sliceIndex, j, endpoint.Hostname, wantEndpoint.Hostname)
		}

		if (endpoint.TargetRef == nil) != (wantEndpoint.TargetRef == nil) {
			t.Errorf("EndpointSlice[%d].Endpoints[%d].TargetRef presence mismatch", sliceIndex, j)
		} else if endpoint.TargetRef != nil && wantEndpoint.TargetRef != nil {
			if endpoint.TargetRef.Kind != wantEndpoint.TargetRef.Kind ||
				endpoint.TargetRef.Name != wantEndpoint.TargetRef.Name ||
				endpoint.TargetRef.Namespace != wantEndpoint.TargetRef.Namespace {
				t.Errorf("EndpointSlice[%d].Endpoints[%d].TargetRef = %v, want %v", sliceIndex, j, endpoint.TargetRef, wantEndpoint.TargetRef)
			}
		}
	}
}

func createTestEndpointSlice(name string, addressType discoveryv1.AddressType,
	ports []discoveryv1.EndpointPort, endpoints []discoveryv1.Endpoint) discoveryv1.EndpointSlice {
	return discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				discoveryv1.LabelServiceName: "test-service",
			},
		},
		AddressType: addressType,
		Ports:       ports,
		Endpoints:   endpoints,
	}
}

func createHTTPPort() discoveryv1.EndpointPort {
	return discoveryv1.EndpointPort{
		Name:     ptr.To("http"),
		Port:     ptr.To[int32](80),
		Protocol: ptr.To(corev1.ProtocolTCP),
	}
}

func createHTTPSPort(port int32) discoveryv1.EndpointPort {
	return discoveryv1.EndpointPort{
		Name:     ptr.To("https"),
		Port:     ptr.To(port),
		Protocol: ptr.To(corev1.ProtocolTCP),
	}
}

func createPlainPort(port int32) discoveryv1.EndpointPort {
	return discoveryv1.EndpointPort{
		Port:     ptr.To(port),
		Protocol: ptr.To(corev1.ProtocolTCP),
	}
}

func createReadyEndpoint(address string) discoveryv1.Endpoint {
	return discoveryv1.Endpoint{
		Addresses: []string{address},
		Conditions: discoveryv1.EndpointConditions{
			Ready: ptr.To(true),
		},
	}
}

func createNotReadyEndpoint(address string) discoveryv1.Endpoint {
	return discoveryv1.Endpoint{
		Addresses: []string{address},
		Conditions: discoveryv1.EndpointConditions{
			Ready: ptr.To(false),
		},
	}
}

func createEndpointWithHostname(address, hostname string) discoveryv1.Endpoint {
	return discoveryv1.Endpoint{
		Addresses: []string{address},
		Conditions: discoveryv1.EndpointConditions{
			Ready: ptr.To(true),
		},
		Hostname: ptr.To(hostname),
		TargetRef: &corev1.ObjectReference{
			Kind:      "Pod",
			Name:      hostname,
			Namespace: "default",
		},
	}
}
