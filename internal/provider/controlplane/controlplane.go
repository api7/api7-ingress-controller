// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"fmt"

	"github.com/api7/gopkg/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/controlplane/translator"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
)

type dashboardProvider struct {
	translator *translator.Translator
	c          dashboard.Dashboard
}

//nolint:unused
func NewDashboard() (provider.Provider, error) {
	control, err := dashboard.NewClient()
	if err != nil {
		return nil, err
	}

	if err := control.AddCluster(context.TODO(), &dashboard.ClusterOptions{
		Name: "default",
		Labels: map[string]string{
			"k8s/controller-name": config.ControllerConfig.ControllerName,
		},
		ControllerName: config.ControllerConfig.ControllerName,
		SyncCache:      true,
	}); err != nil {
		return nil, err
	}

	return &dashboardProvider{
		translator: &translator.Translator{},
		c:          control,
	}, nil
}

func (d *dashboardProvider) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	var result *translator.TranslateResult
	var err error
	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, obj.DeepCopy())
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, obj.DeepCopy())
	}
	if err != nil {
		return err
	}
	// TODO: support diff resources
	name := "default"
	for _, service := range result.Services {
		if _, err := d.c.Cluster(name).Service().Update(ctx, service); err != nil {
			return err
		}
	}
	for _, route := range result.Routes {
		if _, err := d.c.Cluster(name).Route().Update(ctx, route); err != nil {
			return err
		}
	}
	for _, ssl := range result.SSL {
		// to avoid duplication
		ssl.Snis = arrayUniqueElements(ssl.Snis, []string{})
		if len(ssl.Snis) == 1 && ssl.Snis[0] == "*" {
			log.Warnf("wildcard hostname is not allowed in ssl object. Skipping SSL creation for %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
			return nil
		}
		ssl.Snis = removeWildcard(ssl.Snis)
		oldssl, err := d.c.Cluster(name).SSL().Get(ctx, ssl.Cert)
		if err != nil || oldssl == nil {
			if _, err := d.c.Cluster(name).SSL().Create(ctx, ssl); err != nil {
				return fmt.Errorf("failed to create ssl for sni %+v: %w", ssl.Snis, err)
			}
		} else {
			// array union is done to avoid host duplication
			ssl.Snis = arrayUniqueElements(ssl.Snis, oldssl.Snis)
			if _, err := d.c.Cluster(name).SSL().Update(ctx, ssl); err != nil {
				return fmt.Errorf("failed to update ssl for sni %+v: %w", ssl.Snis, err)
			}
		}
	}
	return nil
}

func removeWildcard(snis []string) []string {
	newSni := make([]string, 0)
	for _, sni := range snis {
		if sni != "*" {
			newSni = append(newSni, sni)
		}
	}
	return newSni
}

func arrayUniqueElements(arr1 []string, arr2 []string) []string {
	// return a union of elements from both array
	presentEle := make(map[string]bool)
	newArr := make([]string, 0)
	for _, ele := range arr1 {
		if !presentEle[ele] {
			presentEle[ele] = true
			newArr = append(newArr, ele)
		}
	}
	for _, ele := range arr2 {
		if !presentEle[ele] {
			presentEle[ele] = true
			newArr = append(newArr, ele)
		}
	}
	return newArr
}

func (d *dashboardProvider) Delete(ctx context.Context, obj client.Object) error {
	clusters := d.c.ListClusters()
	kindLabel := dashboard.ListByKindLabelOptions{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	for _, cluster := range clusters {
		switch obj.(type) {
		case *gatewayv1.Gateway:
			ssls, _ := cluster.SSL().List(ctx, dashboard.ListOptions{
				From:      dashboard.ListFromCache,
				KindLabel: kindLabel,
			})
			for _, ssl := range ssls {
				if err := cluster.SSL().Delete(ctx, ssl); err != nil {
					return err
				}
			}
		case *gatewayv1.HTTPRoute:
			routes, _ := cluster.Route().List(ctx, dashboard.ListOptions{
				From:      dashboard.ListFromCache,
				KindLabel: kindLabel,
			})

			for _, route := range routes {
				if err := cluster.Route().Delete(ctx, route); err != nil {
					return err
				}
			}

			services, _ := cluster.Service().List(ctx, dashboard.ListOptions{
				From:      dashboard.ListFromCache,
				KindLabel: kindLabel,
			})

			for _, service := range services {
				if err := cluster.Service().Delete(ctx, service); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (d *dashboardProvider) Sync(ctx context.Context) error {
	return nil
}

func (d *dashboardProvider) Start(ctx context.Context) error {
	return nil
}
