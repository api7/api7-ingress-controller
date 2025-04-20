package controller

import (
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

type PolicyTargetKey struct {
	NsName    types.NamespacedName
	GroupKind schema.GroupKind
}

func (p PolicyTargetKey) String() string {
	return p.NsName.String() + "/" + p.GroupKind.String()
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
				GroupKind: schema.GroupKind{Group: "", Kind: "Service"},
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
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, policy.DeepCopy())
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
		if c.AncestorRef.Name == ancestorStatus.AncestorRef.Name &&
			ptr.Equal(c.AncestorRef.Namespace, ancestorStatus.AncestorRef.Namespace) &&
			ptr.Equal(c.AncestorRef.Group, ancestorStatus.AncestorRef.Group) &&
			ptr.Equal(c.AncestorRef.Kind, ancestorStatus.AncestorRef.Kind) &&
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

func DeleteAncestors(status *v1alpha1.PolicyStatus, parentRefs []gatewayv1.ParentReference) bool {
	var length = len(status.Ancestors)
	for _, parentRef := range parentRefs {
		status.Ancestors = slices.DeleteFunc(status.Ancestors, func(status gatewayv1alpha2.PolicyAncestorStatus) bool {
			return parentRefValueEqual(parentRef, status.AncestorRef)
		})
	}
	return length != len(status.Ancestors)
}

func parentRefValueEqual(a, b gatewayv1.ParentReference) bool {
	return ptr.Equal(a.Group, b.Group) &&
		ptr.Equal(a.Kind, b.Kind) &&
		ptr.Equal(a.Namespace, b.Namespace) &&
		a.Name == b.Name
}
