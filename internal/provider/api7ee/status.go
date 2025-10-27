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

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
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

func (d *api7eeProvider) updateStatus(nnk types.NamespacedNameKind, condition metav1.Condition) {
	switch nnk.Kind {
	case types.KindApisixRoute:
		d.updateApisixCRDStatus(nnk, &apiv2.ApisixRoute{}, condition)
	case types.KindApisixGlobalRule:
		d.updateApisixCRDStatus(nnk, &apiv2.ApisixGlobalRule{}, condition)
	case types.KindApisixTls:
		d.updateApisixCRDStatus(nnk, &apiv2.ApisixTls{}, condition)
	case types.KindApisixConsumer:
		d.updateApisixCRDStatus(nnk, &apiv2.ApisixConsumer{}, condition)
	case types.KindHTTPRoute:
		d.updateGatewayRouteStatus(nnk, &gatewayv1.HTTPRoute{}, condition)
	case types.KindTCPRoute:
		log.Debugw("updating TCPRoute status",
			zap.Any("parentRefs", d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)))
		d.updateGatewayRouteStatus(nnk, &gatewayv1alpha2.TCPRoute{}, condition)
	case types.KindUDPRoute:
		log.Debugw("updating UDPRoute status",
			zap.Any("parentRefs", d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)))
		d.updateGatewayRouteStatus(nnk, &gatewayv1alpha2.UDPRoute{}, condition)
	case types.KindGRPCRoute:
		log.Debugw("updating GRPCRoute status",
			zap.Any("parentRefs", d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)))
		d.updateGatewayRouteStatus(nnk, &gatewayv1.GRPCRoute{}, condition)
	}
}

// updateApisixCRDStatus updates the status of APISIX CRD resources.
func (d *api7eeProvider) updateApisixCRDStatus(
	nnk types.NamespacedNameKind,
	resource client.Object,
	condition metav1.Condition,
) {
	d.updater.Update(status.Update{
		NamespacedName: nnk.NamespacedName(),
		Resource:       resource,
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
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
			default:
				return obj
			}
		}),
	})
}

// updateGatewayRouteStatus updates the status of Gateway API route resources.
func (d *api7eeProvider) updateGatewayRouteStatus(
	nnk types.NamespacedNameKind,
	resource client.Object,
	condition metav1.Condition,
) {
	parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
	gatewayRefs := d.extractGatewayRefs(parentRefs)

	d.updater.Update(status.Update{
		NamespacedName: nnk.NamespacedName(),
		Resource:       resource,
		Mutator:        status.MutatorFunc(d.createRouteMutator(gatewayRefs, condition)),
	})
}

// extractGatewayRefs extracts gateway references from parent references.
func (d *api7eeProvider) extractGatewayRefs(parentRefs []types.NamespacedNameKind) map[types.NamespacedNameKind]struct{} {
	gatewayRefs := map[types.NamespacedNameKind]struct{}{}
	for _, parentRef := range parentRefs {
		if parentRef.Kind == types.KindGateway {
			gatewayRefs[parentRef] = struct{}{}
		}
	}
	return gatewayRefs
}

// createRouteMutator creates a mutator function for updating route parent status.
func (d *api7eeProvider) createRouteMutator(
	gatewayRefs map[types.NamespacedNameKind]struct{},
	condition metav1.Condition,
) func(obj client.Object) client.Object {
	return func(obj client.Object) client.Object {
		switch route := obj.(type) {
		case *gatewayv1.HTTPRoute:
			cp := route.DeepCopy()
			d.updateParentStatus(&cp.Status.RouteStatus, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1alpha2.TCPRoute:
			cp := route.DeepCopy()
			d.updateParentStatus(&cp.Status.RouteStatus, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1alpha2.UDPRoute:
			cp := route.DeepCopy()
			d.updateParentStatus(&cp.Status.RouteStatus, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		case *gatewayv1.GRPCRoute:
			cp := route.DeepCopy()
			d.updateParentStatus(&cp.Status.RouteStatus, cp.GetNamespace(), gatewayRefs, condition)
			return cp
		default:
			return obj
		}
	}
}

// updateParentStatus updates the parent status for route resources.
func (d *api7eeProvider) updateParentStatus(
	routeStatus *gatewayv1.RouteStatus,
	defaultNamespace string,
	gatewayRefs map[types.NamespacedNameKind]struct{},
	condition metav1.Condition,
) {
	for i, ref := range routeStatus.Parents {
		if !d.shouldUpdateParentRef(ref.ParentRef, defaultNamespace, gatewayRefs) {
			continue
		}
		ref.Conditions = cutils.MergeCondition(ref.Conditions, condition)
		routeStatus.Parents[i] = ref
	}
}

// shouldUpdateParentRef checks if a parent reference should be updated.
func (d *api7eeProvider) shouldUpdateParentRef(
	parentRef gatewayv1.ParentReference,
	defaultNamespace string,
	gatewayRefs map[types.NamespacedNameKind]struct{},
) bool {
	if parentRef.Kind != nil && *parentRef.Kind != types.KindGateway {
		return false
	}

	ns := defaultNamespace
	if parentRef.Namespace != nil {
		ns = string(*parentRef.Namespace)
	}

	nnk := types.NamespacedNameKind{
		Name:      string(parentRef.Name),
		Namespace: ns,
		Kind:      types.KindGateway,
	}

	_, exists := gatewayRefs[nnk]
	return exists
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
		log.Errorw("failed to get resources from store", zap.String("configName", configName), zap.Error(err))
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
		log.Errorw("failed to list global rules", zap.String("configName", configName), zap.Error(err))
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
			log.Errorw("failed to get resource label",
				zap.String("configName", configName),
				zap.String("resourceType", status.Event.ResourceType),
				zap.String("id", id),
				zap.Error(err),
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
