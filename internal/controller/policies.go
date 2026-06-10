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

package controller

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

type PolicyTargetKey struct {
	NsName    types.NamespacedName
	GroupKind schema.GroupKind
}

func (p PolicyTargetKey) String() string {
	return p.NsName.String() + "/" + p.GroupKind.String()
}

func BackendTrafficPolicyPredicateFunc(channel chan event.GenericEvent) predicate.Predicate {
	return predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj, ok := e.ObjectOld.(*v1alpha1.BackendTrafficPolicy)
			newObj, ok2 := e.ObjectNew.(*v1alpha1.BackendTrafficPolicy)
			if !ok || !ok2 {
				return false
			}
			oldRefs := oldObj.Spec.TargetRefs
			newRefs := newObj.Spec.TargetRefs

			oldRefMap := make(map[string]v1alpha1.BackendPolicyTargetReferenceWithSectionName)
			for _, ref := range oldRefs {
				key := fmt.Sprintf("%s/%s/%s", ref.Group, ref.Kind, ref.Name)
				oldRefMap[key] = ref
			}

			for _, ref := range newRefs {
				key := fmt.Sprintf("%s/%s/%s", ref.Group, ref.Kind, ref.Name)
				delete(oldRefMap, key)
			}
			if len(oldRefMap) > 0 {
				targetRefs := make([]v1alpha1.BackendPolicyTargetReferenceWithSectionName, 0, len(oldRefs))
				for _, ref := range oldRefMap {
					targetRefs = append(targetRefs, ref)
				}
				dump := oldObj.DeepCopy()
				dump.Spec.TargetRefs = targetRefs
				channel <- event.GenericEvent{
					Object: dump,
				}
			}
			return true
		},
	}
}

func ProcessBackendTrafficPolicy(
	c client.Client,
	log logr.Logger,
	tctx *provider.TranslateContext,
) {
	conflicts := map[string]*v1alpha1.BackendTrafficPolicy{}
	servicePortNameMap := map[string]bool{}
	policyMap := map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy{}
	for _, service := range tctx.Services {
		backendTrafficPolicyList := &v1alpha1.BackendTrafficPolicyList{}
		if err := c.List(tctx, backendTrafficPolicyList,
			client.MatchingFields{
				indexer.PolicyTargetRefs: indexer.GenIndexKeyWithGK("", "Service", service.Namespace, service.Name),
			},
		); err != nil {
			log.Error(err, "failed to list BackendTrafficPolicy for Service")
			continue
		}
		if len(backendTrafficPolicyList.Items) == 0 {
			continue
		}
		for _, port := range service.Spec.Ports {
			key := fmt.Sprintf("%s/%s/%s", service.Namespace, service.Name, port.Name)
			servicePortNameMap[key] = true
		}

		for _, p := range backendTrafficPolicyList.Items {
			policyMap[types.NamespacedName{
				Name:      p.Name,
				Namespace: p.Namespace,
			}] = p.DeepCopy()
		}
	}

	for _, p := range policyMap {
		policy := p.DeepCopy()
		targetRefs := policy.Spec.TargetRefs
		updated := false
		for _, targetRef := range targetRefs {
			sectionName := targetRef.SectionName
			key := PolicyTargetKey{
				NsName:    types.NamespacedName{Namespace: p.GetNamespace(), Name: string(targetRef.Name)},
				GroupKind: schema.GroupKind{Group: "", Kind: internaltypes.KindService},
			}
			condition := NewPolicyCondition(policy.Generation, true, "Policy has been accepted")
			if sectionName != nil && !servicePortNameMap[fmt.Sprintf("%s/%s/%s", policy.Namespace, string(targetRef.Name), *sectionName)] {
				condition = NewPolicyCondition(policy.Generation, false, fmt.Sprintf("No section name %s found in Service %s/%s", *sectionName, policy.Namespace, targetRef.Name))
				processPolicyStatus(policy, tctx, condition, &updated)
				continue
			}
			if _, ok := conflicts[key.String()]; ok {
				condition = NewPolicyConflictCondition(policy.Generation, fmt.Sprintf("Unable to target Service %s/%s, because it conflicts with another BackendTrafficPolicy", policy.Namespace, targetRef.Name))
				processPolicyStatus(policy, tctx, condition, &updated)
				continue
			}
			conflicts[key.String()] = policy
			processPolicyStatus(policy, tctx, condition, &updated)
		}
		if updated {
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, status.Update{
				NamespacedName: utils.NamespacedName(policy),
				Resource:       policy.DeepCopy(),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					cp := obj.(*v1alpha1.BackendTrafficPolicy).DeepCopy()
					cp.Status = policy.Status
					return cp
				}),
			})
		}
	}
	for _, policy := range conflicts {
		tctx.BackendTrafficPolicies[types.NamespacedName{
			Name:      policy.Name,
			Namespace: policy.Namespace,
		}] = policy
	}
}

func processPolicyStatus(policy *v1alpha1.BackendTrafficPolicy,
	tctx *provider.TranslateContext,
	condition metav1.Condition,
	updated *bool,
) {
	if ok := SetAncestors(&policy.Status, tctx.RouteParentRefs, condition); ok {
		*updated = true
	}
}

func SetAncestors(status *v1alpha1.PolicyStatus, parentRefs []gatewayv1.ParentReference, condition metav1.Condition) bool {
	updated := false
	for _, parent := range parentRefs {
		ancestorStatus := gatewayv1alpha2.PolicyAncestorStatus{
			AncestorRef:    parent,
			Conditions:     []metav1.Condition{condition},
			ControllerName: gatewayv1alpha2.GatewayController(config.ControllerConfig.ControllerName),
		}
		if SetAncestorStatus(status, ancestorStatus) {
			updated = true
		}
	}
	return updated
}

func SetAncestorStatus(status *v1alpha1.PolicyStatus, ancestorStatus gatewayv1alpha2.PolicyAncestorStatus) bool {
	if len(ancestorStatus.Conditions) == 0 {
		return false
	}
	condition := ancestorStatus.Conditions[0]
	for _, c := range status.Ancestors {
		if parentRefValueEqual(ancestorStatus.AncestorRef, c.AncestorRef) &&
			c.ControllerName == ancestorStatus.ControllerName {
			if !VerifyConditions(&c.Conditions, condition) {
				return false
			}
			meta.SetStatusCondition(&c.Conditions, condition)
			return true
		}
	}
	status.Ancestors = append(status.Ancestors, ancestorStatus)
	return true
}

func parentRefValueEqual(a, b gatewayv1.ParentReference) bool {
	return ptr.Equal(a.Group, b.Group) &&
		ptr.Equal(a.Kind, b.Kind) &&
		ptr.Equal(a.Namespace, b.Namespace) &&
		a.Name == b.Name
}

// ProcessL4RoutePolicy finds L4RoutePolicy resources that target the given L4 route
// (identified by namespace, name, and kind), resolves conflicts deterministically,
// populates tctx.L4RoutePolicies with the winning policy, and queues status updates.
func ProcessL4RoutePolicy(
	c client.Client,
	log logr.Logger,
	tctx *provider.TranslateContext,
	routeNamespace, routeName, routeKind string,
) {
	var list v1alpha1.L4RoutePolicyList
	key := indexer.GenIndexKeyWithGK(gatewayv1alpha2.GroupName, routeKind, routeNamespace, routeName)
	if err := c.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		log.Error(err, "failed to list L4RoutePolicy", "namespace", routeNamespace, "name", routeName, "kind", routeKind)
		return
	}
	if len(list.Items) == 0 {
		return
	}

	// Deterministic conflict resolution: oldest creationTimestamp wins; tie-break by namespace/name.
	sort.Slice(list.Items, func(i, j int) bool {
		ti := list.Items[i].CreationTimestamp.Time
		tj := list.Items[j].CreationTimestamp.Time
		if ti.Equal(tj) {
			ki := list.Items[i].Namespace + "/" + list.Items[i].Name
			kj := list.Items[j].Namespace + "/" + list.Items[j].Name
			return ki < kj
		}
		return ti.Before(tj)
	})

	winner := list.Items[0].DeepCopy()
	tctx.L4RoutePolicies[types.NamespacedName{Namespace: winner.Namespace, Name: winner.Name}] = winner

	for i := range list.Items {
		policy := list.Items[i]
		var condition metav1.Condition
		if i == 0 {
			condition = metav1.Condition{
				Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: policy.GetGeneration(),
				LastTransitionTime: metav1.Now(),
				Reason:             string(gatewayv1alpha2.PolicyReasonAccepted),
				Message:            "Policy has been accepted",
			}
		} else {
			condition = metav1.Condition{
				Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.GetGeneration(),
				LastTransitionTime: metav1.Now(),
				Reason:             string(gatewayv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflicts with L4RoutePolicy %s/%s which was created earlier", winner.Namespace, winner.Name),
			}
		}

		if updated := SetAncestors(&policy.Status, tctx.RouteParentRefs, condition); updated {
			// Resource must be a separate copy from the object captured by the Mutator:
			// the status updater calls client.Get into Resource, overwriting it with the
			// server state. The Mutator reads policy.Status, which keeps the ancestors set above.
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, status.Update{
				NamespacedName: utils.NamespacedName(&policy),
				Resource:       policy.DeepCopy(),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					cp := obj.(*v1alpha1.L4RoutePolicy).DeepCopy()
					cp.Status = policy.Status
					return cp
				}),
			})
		}
	}
}

// updateL4RoutePolicyStatusOnDeleting clears ancestor status entries for L4RoutePolicy
// resources that target the deleted route.
func updateL4RoutePolicyStatusOnDeleting(ctx context.Context, c client.Client, updater status.Updater, log logr.Logger, nn types.NamespacedName, routeKind string) {
	var list v1alpha1.L4RoutePolicyList
	key := indexer.GenIndexKeyWithGK(gatewayv1alpha2.GroupName, routeKind, nn.Namespace, nn.Name)
	if err := c.List(ctx, &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		log.Error(err, "failed to list L4RoutePolicy on route deletion", "namespace", nn.Namespace, "name", nn.Name)
		return
	}
	for _, policy := range list.Items {
		updateL4RoutePolicyDeleteAncestors(updater, policy)
	}
}

func updateL4RoutePolicyDeleteAncestors(updater status.Updater, policy v1alpha1.L4RoutePolicy) {
	if len(policy.Status.Ancestors) == 0 {
		return
	}
	// status.ancestors is a required field; use an empty slice (serializes to [])
	// rather than nil (serializes to null), which the CRD schema rejects.
	policy.Status.Ancestors = []gatewayv1alpha2.PolicyAncestorStatus{}
	updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(&policy),
		Resource:       policy.DeepCopy(),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp := obj.(*v1alpha1.L4RoutePolicy).DeepCopy()
			cp.Status = policy.Status
			return cp
		}),
	})
}
