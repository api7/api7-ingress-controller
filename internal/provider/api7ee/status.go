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

package api7ee

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	cutils "github.com/apache/apisix-ingress-controller/internal/controller/utils"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type statusMutator struct {
	resource    client.Object
	mutatorFunc status.MutatorFunc
}
// handleStatusUpdate updates resource conditions based on the latest sync results.
//
// It maintains a history of failed resources in d.statusUpdateMap.
//
// For resources in the current failure map (statusUpdateMap), it marks them as failed.
// For resources that exist only in the previous failure history (i.e. not in this sync's failures),
// it marks them as accepted (success).
func (d *api7eeProvider) handleStatusUpdate(statusUpdateMap map[types.NamespacedNameKind][]string) {
	// Mark all resources in the current failure set as failed.
	for nnk, msgs := range statusUpdateMap {
		d.updateStatus(nnk, cutils.NewConditionTypeAccepted(
			apiv2.ConditionReasonSyncFailed,
			false,
			0,
			strings.Join(msgs, "; "),
		))
	}

	// Mark resources that exist only in the previous failure history as successful.
	for nnk := range d.statusUpdateMap {
		if _, ok := statusUpdateMap[nnk]; !ok {
			d.updateStatus(nnk, cutils.NewConditionTypeAccepted(
				apiv2.ConditionReasonAccepted,
				true,
				0,
				"",
			))
		}
	}
	// Update the failure history with the current failure set.
	d.statusUpdateMap = statusUpdateMap
}

//nolint:gocyclo
func (d *api7eeProvider) updateStatus(nnk types.NamespacedNameKind, condition metav1.Condition) {
	// Simple APISIX CRDs
	if mutator := d.getApisixCRDMutator(nnk.Kind, condition); mutator != nil {
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       mutator.resource,
			Mutator:        mutator.mutatorFunc,
		})
		return
	}

	// Gateway API Routes
	if d.isGatewayRoute(nnk.Kind) {
		d.updateGatewayRoute(nnk, condition)
	}
}

func (d *api7eeProvider) getApisixCRDMutator(kind string, condition metav1.Condition) *statusMutator {
	makeApisixMutator := func(resource client.Object) *statusMutator {
		return &statusMutator{
			resource: resource,
			mutatorFunc: func(obj client.Object) client.Object {
				switch v := obj.(type) {
				case *apiv2.ApisixRoute:
					cp := v.DeepCopy()
					cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
					return cp
				case *apiv2.ApisixGlobalRule:
					cp := v.DeepCopy()
					cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
					return cp
				case *apiv2.ApisixTls:
					cp := v.DeepCopy()
					cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
					return cp
				case *apiv2.ApisixConsumer:
					cp := v.DeepCopy()
					cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
					return cp
				}
				return obj
			},
		}
	}

	switch kind {
	case types.KindApisixRoute:
		return makeApisixMutator(&apiv2.ApisixRoute{})
	case types.KindApisixGlobalRule:
		return makeApisixMutator(&apiv2.ApisixGlobalRule{})
	case types.KindApisixTls:
		return makeApisixMutator(&apiv2.ApisixTls{})
	case types.KindApisixConsumer:
		return makeApisixMutator(&apiv2.ApisixConsumer{})
	}
	return nil
}

func (d *api7eeProvider) isGatewayRoute(kind string) bool {
	return kind == types.KindHTTPRoute || kind == types.KindUDPRoute ||
		kind == types.KindTCPRoute || kind == types.KindGRPCRoute ||
		kind == types.KindTLSRoute
}

func (d *api7eeProvider) updateGatewayRoute(nnk types.NamespacedNameKind, condition metav1.Condition) {
	parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
	if nnk.Kind != types.KindHTTPRoute && nnk.Kind != types.KindGRPCRoute {
		d.log.V(1).Info(fmt.Sprintf("updating %s status", nnk.Kind), "parentRefs", parentRefs)
	}

	gatewayRefs := d.filterGatewayRefs(parentRefs)

	resource := d.getGatewayResource(nnk.Kind)
	if resource == nil {
		return
	}

	d.updater.Update(status.Update{
		NamespacedName: nnk.NamespacedName(),
		Resource:       resource,
		Mutator:        d.makeGatewayMutator(gatewayRefs, condition),
	})
}

func (d *api7eeProvider) filterGatewayRefs(parentRefs []types.NamespacedNameKind) map[types.NamespacedNameKind]struct{} {
	gatewayRefs := map[types.NamespacedNameKind]struct{}{}
	for _, parentRef := range parentRefs {
		if parentRef.Kind == types.KindGateway {
			gatewayRefs[parentRef] = struct{}{}
		}
	}
	return gatewayRefs
}

func (d *api7eeProvider) getGatewayResource(kind string) client.Object {
	switch kind {
	case types.KindHTTPRoute:
		return &gatewayv1.HTTPRoute{}
	case types.KindUDPRoute:
		return &gatewayv1alpha2.UDPRoute{}
	case types.KindTCPRoute:
		return &gatewayv1alpha2.TCPRoute{}
	case types.KindGRPCRoute:
		return &gatewayv1.GRPCRoute{}
	case types.KindTLSRoute:
		return &gatewayv1alpha2.TLSRoute{}
	}
	return nil
}

func (d *api7eeProvider) makeGatewayMutator(gatewayRefs map[types.NamespacedNameKind]struct{}, condition metav1.Condition) status.MutatorFunc {
	return func(obj client.Object) client.Object {
		switch route := obj.(type) {
		case *gatewayv1.HTTPRoute:
			cp := route.DeepCopy()
			d.updateRouteParentStatuses(&cp.Status.Parents, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1alpha2.UDPRoute:
			cp := route.DeepCopy()
			d.updateRouteParentStatuses(&cp.Status.Parents, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1alpha2.TCPRoute:
			cp := route.DeepCopy()
			d.updateRouteParentStatuses(&cp.Status.Parents, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1.GRPCRoute:
			cp := route.DeepCopy()
			d.updateRouteParentStatuses(&cp.Status.Parents, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1alpha2.TLSRoute:
			cp := route.DeepCopy()
			d.updateRouteParentStatuses(&cp.Status.Parents, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		}
		return obj
	}
}

func (d *api7eeProvider) updateRouteParentStatuses(parents *[]gatewayv1.RouteParentStatus, routeNamespace string, gatewayRefs map[types.NamespacedNameKind]struct{}, condition metav1.Condition) {
	for i, ref := range *parents {
		ns := routeNamespace
		if ref.ParentRef.Namespace != nil {
			ns = string(*ref.ParentRef.Namespace)
		}
		if ref.ParentRef.Kind == nil || *ref.ParentRef.Kind == types.KindGateway {
			nnk := types.NamespacedNameKind{
				Name:      string(ref.ParentRef.Name),
				Namespace: ns,
				Kind:      types.KindGateway,
			}
			if _, ok := gatewayRefs[nnk]; ok {
				ref.Conditions = cutils.MergeCondition(ref.Conditions, condition)
				(*parents)[i] = ref
			}
		}
	}
}

func (d *api7eeProvider) resolveADCExecutionErrors(
	statusesMap map[string]types.ADCExecutionErrors,
) map[types.NamespacedNameKind][]string {
	statusUpdateMap := map[types.NamespacedNameKind][]string{}
	for configName, execErrors := range statusesMap {
		for _, execErr := range execErrors.Errors {
			for _, failedStatus := range execErr.FailedErrors {
				if len(failedStatus.FailedStatuses) == 0 {
					d.handleEmptyFailedStatuses(configName, failedStatus, statusUpdateMap)
				} else {
					d.handleDetailedFailedStatuses(configName, failedStatus, statusUpdateMap)
				}
			}
		}
	}

	return statusUpdateMap
}

func (d *api7eeProvider) handleEmptyFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	resource, err := d.client.GetResources(configName)
	if err != nil {
		d.log.Error(err, "failed to get resources from store", "configName", configName)
		return
	}

	for _, obj := range resource.Services {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	for _, obj := range resource.Consumers {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	for _, obj := range resource.SSLs {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	globalRules, err := d.client.ListGlobalRules(configName)
	if err != nil {
		d.log.Error(err, "failed to list global rules", "configName", configName)
		return
	}
	for _, rule := range globalRules {
		d.addResourceToStatusUpdateMap(rule.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}
}

func (d *api7eeProvider) handleDetailedFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	for _, status := range failedStatus.FailedStatuses {
		id := status.Event.ResourceID
		labels, err := d.client.GetResourceLabel(configName, status.Event.ResourceType, id)
		if err != nil {
			d.log.Error(err, "failed to get resource label",
				"configName", configName,
				"resourceType", status.Event.ResourceType,
				"id", id,
			)
			continue
		}
		d.addResourceToStatusUpdateMap(
			labels,
			fmt.Sprintf("ServerAddr: %s, Error: %s", failedStatus.ServerAddr, status.Reason),
			statusUpdateMap,
		)
	}
}

func (d *api7eeProvider) addResourceToStatusUpdateMap(
	labels map[string]string,
	msg string,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	statusKey := types.NamespacedNameKind{
		Name:      labels[label.LabelName],
		Namespace: labels[label.LabelNamespace],
		Kind:      labels[label.LabelKind],
	}
	statusUpdateMap[statusKey] = append(statusUpdateMap[statusKey], msg)
}
