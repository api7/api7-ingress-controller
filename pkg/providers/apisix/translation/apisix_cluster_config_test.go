// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2 "github.com/api7/api7-ingress-controller/pkg/kube/apisix/apis/config/v2"
)

func TestTranslateClusterConfigV2(t *testing.T) {
	tr := &translator{}

	acc := &configv2.ApisixClusterConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "qa-apisix",
		},
		Spec: configv2.ApisixClusterConfigSpec{
			Monitoring: &configv2.ApisixClusterMonitoringConfig{
				Skywalking: configv2.ApisixClusterSkywalkingConfig{
					Enable:      true,
					SampleRatio: 0.5,
				},
			},
		},
	}
	gr, err := tr.TranslateClusterConfigV2(acc)
	assert.Nil(t, err, "translating ApisixClusterConfigV2")
	assert.Equal(t, gr.ID, "skywalking", "checking global_rule id")
	assert.Len(t, gr.Plugins, 1)
	assert.Equal(t, gr.Plugins["skywalking"], &skywalkingPluginConfig{SampleRatio: 0.5})
}
