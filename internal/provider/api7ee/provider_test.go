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

package api7ee

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
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
