// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package k8s

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"

	apisixprovider "github.com/api7/api7-ingress-controller/pkg/providers/apisix"
	ingressprovider "github.com/api7/api7-ingress-controller/pkg/providers/ingress"
	"github.com/api7/api7-ingress-controller/pkg/providers/k8s/configmap"
	"github.com/api7/api7-ingress-controller/pkg/providers/k8s/endpoint"
	"github.com/api7/api7-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/api7/api7-ingress-controller/pkg/providers/translation"
	providertypes "github.com/api7/api7-ingress-controller/pkg/providers/types"
	"github.com/api7/api7-ingress-controller/pkg/providers/utils"
)

var _ Provider = (*k8sProvider)(nil)

type Provider interface {
	providertypes.Provider
}

type k8sProvider struct {
	secretController *secretController
	endpoint         endpoint.Provider
	configmap        configmap.Provider

	secretInformer cache.SharedIndexInformer
}

func NewProvider(common *providertypes.Common, translator translation.Translator,
	namespaceProvider namespace.WatchingNamespaceProvider,
	apisixProvider apisixprovider.Provider, ingressProvider ingressprovider.Provider) (Provider, error) {
	var err error
	provider := &k8sProvider{}

	kubeFactory := common.KubeClient.NewSharedIndexInformerFactory()
	provider.secretInformer = kubeFactory.Core().V1().Secrets().Informer()

	provider.endpoint, err = endpoint.NewProvider(common, translator, namespaceProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init endpoint provider")
	}

	provider.secretController = newSecretController(common, namespaceProvider, apisixProvider, ingressProvider)

	provider.configmap, err = configmap.NewProvider(common)
	if err != nil {
		return nil, errors.Wrap(err, "failed, to init configmap provider")
	}

	return provider, nil
}

func (p *k8sProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.secretController.run(ctx)
	})
	e.Add(func() {
		p.endpoint.Run(ctx)
	})

	e.Add(func() {
		p.configmap.Run(ctx)
	})

	e.Wait()
}
