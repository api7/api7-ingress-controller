package controller

import (
	"context"
	"encoding/json"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

func (r *HTTPRouteReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	// list HTTPRoutePolices which sectionName is not specified
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(gatewayv1.GroupName, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName())
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	}

	var conflicts = make(map[types.NamespacedName]v1alpha1.HTTPRoutePolicy)
	for _, rule := range httpRoute.Spec.Rules {
		var policies = findPolicyWhichTargetRefTheRule(rule.Name, "HTTPRoute", list)
		if conflict := isPoliciesConflict(policies); conflict {
			for _, policy := range policies {
				namespacedName := types.NamespacedName{Namespace: policy.GetNamespace(), Name: policy.GetName()}
				conflicts[namespacedName] = policy
			}
		}
	}
	data, _ := json.MarshalIndent(conflicts, "", "  ")
	r.Log.Info("conflicts policies", "data", string(data))

	for i := range list.Items {
		var (
			policy         = list.Items[i]
			namespacedName = types.NamespacedName{Namespace: policy.GetNamespace(), Name: policy.GetName()}

			status  = false
			reason  = string(v1alpha2.PolicyReasonConflicted)
			message = "HTTPRoutePolicy conflict with others target to the HTTPRoute"
		)
		if _, conflict := conflicts[namespacedName]; !conflict {
			status = true
			reason = string(v1alpha2.PolicyReasonAccepted)
			message = ""

			tctx.HTTPRoutePolicies = append(tctx.HTTPRoutePolicies, policy)
		}
		modifyHTTPRoutePolicyStatus(httpRoute.Spec.ParentRefs, &policy, status, reason, message)
		tctx.StatusUpdaters = append(tctx.StatusUpdaters, &policy)
	}

	return nil
}

func (r *IngressReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(networkingv1.GroupName, "Ingress", ingress.GetNamespace(), ingress.GetName())
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	}

	var (
		status  = false
		reason  = string(v1alpha2.PolicyReasonConflicted)
		message = "HTTPRoutePolicy conflict with others target to the Ingress"
	)
	if conflict := isPoliciesConflict(list.Items); !conflict {
		status = true
		reason = string(v1alpha2.PolicyReasonAccepted)
		message = ""

		tctx.HTTPRoutePolicies = list.Items
	}

	for i := range list.Items {
		policy := list.Items[i]
		modifyHTTPRoutePolicyStatus(tctx.RouteParentRefs, &policy, status, reason, message)
		tctx.StatusUpdaters = append(tctx.StatusUpdaters, &policy)
	}

	return nil
}

func modifyHTTPRoutePolicyStatus(parentRefs []gatewayv1.ParentReference, policy *v1alpha1.HTTPRoutePolicy, status bool, reason, message string) {
	condition := metav1.Condition{
		Type:               string(v1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	if !status {
		condition.Status = metav1.ConditionFalse
	}
	_ = SetAncestors(&policy.Status, parentRefs, condition)
}

func isPoliciesConflict(policies []v1alpha1.HTTPRoutePolicy) bool {
	if len(policies) == 0 {
		return false
	}
	priority := policies[0].Spec.Priority
	for _, policy := range policies {
		if !ptr.Equal(policy.Spec.Priority, priority) {
			return true
		}
	}
	return false
}

func findPolicyWhichTargetRefTheRule(ruleName *gatewayv1.SectionName, kind string, list v1alpha1.HTTPRoutePolicyList) (policies []v1alpha1.HTTPRoutePolicy) {
	for _, policy := range list.Items {
		for _, ref := range policy.Spec.TargetRefs {
			if string(ref.Kind) == kind && (ref.SectionName == nil || *ref.SectionName == "" || ptr.Equal(ref.SectionName, ruleName)) {
				policies = append(policies, policy)
				break
			}
		}
	}
	return
}
