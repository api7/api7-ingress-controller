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

package translator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateConsumerV1alpha1_UsesMetadataLabelsWithoutOverwritingControllerLabels(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	consumer := &v1alpha1.Consumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Consumer",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			Labels: map[string]string{
				"team":                 "payments",
				label.LabelName:        "user-value",
				label.LabelResourceKey: "user-resource-key",
			},
		},
	}

	result, err := translator.TranslateConsumerV1alpha1(tctx, consumer)
	require.NoError(t, err)
	require.Len(t, result.Consumers, 1)

	translated := result.Consumers[0]
	require.Equal(t, "payments", translated.Labels["team"])
	require.Equal(t, consumer.Name, translated.Labels[label.LabelName])
	require.Equal(t, "Consumer/default/demo", translated.Labels[label.LabelResourceKey])
}

func TestTranslateConsumerRejectsIncompleteConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name        string
		credentials []v1alpha1.Credential
		plugins     []v1alpha1.Plugin
		want        string
	}{
		{name: "malformed credential", credentials: []v1alpha1.Credential{{Name: "bad", Type: "key-auth", Config: apiextensionsv1.JSON{Raw: []byte(`{"key":`)}}}, want: "credential bad"},
		{name: "non-object credential", credentials: []v1alpha1.Credential{{Name: "bad", Type: "key-auth", Config: apiextensionsv1.JSON{Raw: []byte(`[]`)}}}, want: "credential bad"},
		{name: "malformed plugin", plugins: []v1alpha1.Plugin{{Name: "limit-count", Config: apiextensionsv1.JSON{Raw: []byte(`{"count":`)}}}, want: "plugin limit-count"},
		{name: "non-object plugin", plugins: []v1alpha1.Plugin{{Name: "limit-count", Config: apiextensionsv1.JSON{Raw: []byte(`[]`)}}}, want: "plugin limit-count"},
		{name: "missing plugin secret", plugins: []v1alpha1.Plugin{{Name: "limit-count", SecretRef: &corev1.LocalObjectReference{Name: "missing"}}}, want: "secret default/missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			consumer := &v1alpha1.Consumer{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"}}
			consumer.Spec.Credentials = append([]v1alpha1.Credential{{Name: "valid", Type: "key-auth", Config: apiextensionsv1.JSON{Raw: []byte(`{"key":"test-key"}`)}}}, tc.credentials...)
			consumer.Spec.Plugins = append([]v1alpha1.Plugin{{Name: "response-rewrite", Config: apiextensionsv1.JSON{Raw: []byte(`{"status_code":403}`)}}}, tc.plugins...)
			result, err := NewTranslator(logr.Discard(), "").TranslateConsumerV1alpha1(provider.NewDefaultTranslateContext(context.Background()), consumer)
			require.ErrorContains(t, err, tc.want)
			require.ErrorContains(t, err, "consumer default/demo")
			require.Nil(t, result, "must not publish a partial consumer")
			require.NotContains(t, err.Error(), "test-key")
		})
	}
}

func TestTranslateConsumerPreservesValidConfiguration(t *testing.T) {
	ctx := provider.NewDefaultTranslateContext(context.Background())
	ctx.Secrets[types.NamespacedName{Namespace: "default", Name: "credential"}] = &corev1.Secret{Data: map[string][]byte{"key": []byte("secret-key")}}
	consumer := &v1alpha1.Consumer{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"}}
	consumer.Spec.Credentials = []v1alpha1.Credential{
		{Name: "inline", Type: "key-auth", Config: apiextensionsv1.JSON{Raw: []byte(`{"key":"inline-key"}`)}},
		{Name: "secret", Type: "key-auth", SecretRef: &v1alpha1.SecretReference{Name: "credential"}},
	}
	consumer.Spec.Plugins = []v1alpha1.Plugin{{Name: "response-rewrite", Config: apiextensionsv1.JSON{Raw: []byte(`{"status_code":403}`)}}}
	result, err := NewTranslator(logr.Discard(), "").TranslateConsumerV1alpha1(ctx, consumer)
	require.NoError(t, err)
	require.Len(t, result.Consumers, 1)
	require.Len(t, result.Consumers[0].Credentials, 2)
	require.Equal(t, "inline-key", result.Consumers[0].Credentials[0].Config["key"])
	require.Equal(t, "secret-key", result.Consumers[0].Credentials[1].Config["key"])
	require.Equal(t, float64(403), result.Consumers[0].Plugins["response-rewrite"].(map[string]any)["status_code"])
}
