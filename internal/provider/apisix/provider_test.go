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

package apisix

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func TestDeleteLogsObjectIdentityOnly(t *testing.T) {
	const privateValue = "delete-log-private-value"
	var entries []string
	log := funcr.New(func(prefix, args string) {
		entries = append(entries, prefix+args)
	}, funcr.Options{Verbosity: 10})

	p, err := New(log, nil, nil)
	require.NoError(t, err)
	consumer := &apiv2.ApisixConsumer{
		TypeMeta: metav1.TypeMeta{Kind: "ApisixConsumer", APIVersion: apiv2.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "consumer",
		},
		Spec: apiv2.ApisixConsumerSpec{
			AuthParameter: &apiv2.ApisixConsumerAuthParameter{
				KeyAuth: &apiv2.ApisixConsumerKeyAuth{Value: &apiv2.ApisixConsumerKeyAuthValue{Key: privateValue}},
			},
		},
	}

	require.NoError(t, p.Delete(context.Background(), consumer))

	output := strings.Join(entries, "\n")
	assert.NotContains(t, output, privateValue)
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "consumer")
}

// TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved covers the cost side of route
// ownership: a sync pushes the whole store to every data plane, and reconciles
// for routes this controller never configured are frequent (any EndpointSlice
// event on a shared backend enqueues them), so those must not notify.
func TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved(t *testing.T) {
	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPISIX, time.Second)
	require.NoError(t, err)

	d := &apisixProvider{
		client: cli,
		syncCh: make(chan struct{}, 1),
		log:    logr.Discard(),
	}

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}

	require.NoError(t, d.Delete(context.Background(), route))
	require.Empty(t, d.syncCh, "a route this controller never configured must not trigger a sync")

	cli.ConfigManager.Update(utils.NamespacedNameKind(route), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {Name: "proxy"},
	})

	require.NoError(t, d.Delete(context.Background(), route))
	require.Len(t, d.syncCh, 1, "removing configuration this controller pushed must trigger a sync")
}

func TestUpdateKeepsLastKnownGoodStateWhenL4PolicyCannotRender(t *testing.T) {
	rawProvider, err := New(logr.Discard(), nil, nil)
	require.NoError(t, err)
	d := rawProvider.(*apisixProvider)

	route := &gatewayv1.TCPRoute{
		TypeMeta: metav1.TypeMeta{Kind: "TCPRoute", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "tcp-route",
		},
		Spec: gatewayv1.TCPRouteSpec{Rules: []gatewayv1.TCPRouteRule{{}}},
	}
	gatewayProxy := v1alpha1.GatewayProxy{
		TypeMeta:   metav1.TypeMeta{Kind: "GatewayProxy", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "proxy"},
		Spec: v1alpha1.GatewayProxySpec{Provider: &v1alpha1.GatewayProxyProvider{
			Type: v1alpha1.ProviderTypeControlPlane,
			ControlPlane: &v1alpha1.ControlPlaneProvider{
				Endpoints: []string{"http://apisix:9180"},
				Auth: v1alpha1.ControlPlaneAuth{
					Type:     v1alpha1.AuthTypeAdminKey,
					AdminKey: &v1alpha1.AdminKeyAuth{Value: "key"},
				},
			},
		}},
	}
	configName := utils.NamespacedNameKind(&gatewayProxy).String()
	lastKnownGood := adctypes.NewDefaultService()
	lastKnownGood.Name = "last-known-good"
	lastKnownGood.ID = "last-known-good"
	lastKnownGood.Labels = label.GenLabel(route)
	require.NoError(t, d.client.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{lastKnownGood},
	}, lastKnownGood.Labels))

	policy := &v1alpha1.L4RoutePolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "tcp-policy"},
		Spec: v1alpha1.L4RoutePolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: gatewayv1.GroupName,
					Kind:  "TCPRoute",
					Name:  "tcp-route",
				},
			}},
			Plugins: []v1alpha1.Plugin{{
				Name:   "ip-restriction",
				Config: apiextensionsv1.JSON{Raw: []byte(`[]`)},
			}},
		},
	}
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.GatewayProxies[utils.NamespacedNameKind(&gatewayProxy)] = gatewayProxy
	tctx.L4RoutePolicies[k8stypes.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = policy

	err = d.Update(context.Background(), tctx, route)

	require.Error(t, err)
	assert.Empty(t, d.syncCh)
	resources, getErr := d.client.GetResources(configName)
	require.NoError(t, getErr)
	require.Len(t, resources.Services, 1)
	assert.Equal(t, "last-known-good", resources.Services[0].Name)
}
