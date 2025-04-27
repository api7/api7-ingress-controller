package controller

import (
	"context"
	"slices"

	"github.com/go-logr/logr"
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

func (r *HTTPRouteReconciler) updateHTTPRouteStatusOnDeleting(nn types.NamespacedName) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(gatewayv1.GroupName, "HTTPRoute", nn.Namespace, nn.Name)
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}
	var (
		objs       = make(map[types.NamespacedName]struct{})
		parentRefs []gatewayv1.ParentReference
	)
	// collect all parentRefs
	for _, policy := range list.Items {
		for _, ref := range policy.Spec.TargetRefs {
			var obj = types.NamespacedName{Namespace: policy.GetNamespace(), Name: string(ref.Name)}
			if _, ok := objs[obj]; !ok {
				objs[obj] = struct{}{}

				var httpRoute gatewayv1.HTTPRoute
				if err := r.Get(context.Background(), obj, &httpRoute); err != nil {
					continue
				}
				parentRefs = append(parentRefs, httpRoute.Spec.ParentRefs...)
			}
		}
	}
	// delete AncestorRef which is not exist in the all parentRefs for each policy
	updateDeleteAncestors(r.Client, r.Log, list.Items, parentRefs)

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

func (r *IngressReconciler) updateHTTPRoutePolicyStatusOnDeleting(nn types.NamespacedName) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(networkingv1.GroupName, "Ingress", nn.Namespace, nn.Name)
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}
	var (
		objs       = make(map[types.NamespacedName]struct{})
		parentRefs []gatewayv1.ParentReference
	)
	// collect all parentRefs
	for _, policy := range list.Items {
		for _, ref := range policy.Spec.TargetRefs {
			var obj = types.NamespacedName{Namespace: policy.GetNamespace(), Name: string(ref.Name)}
			if _, ok := objs[obj]; !ok {
				objs[obj] = struct{}{}

				var ingress networkingv1.Ingress
				if err := r.Get(context.Background(), obj, &ingress); err != nil {
					continue
				}
				ingressClass, err := r.getIngressClass(&ingress)
				if err != nil {
					continue
				}
				parentRefs = append(parentRefs, gatewayv1.ParentReference{
					Group: ptr.To(gatewayv1.Group(ingressClass.GroupVersionKind().Group)),
					Kind:  ptr.To(gatewayv1.Kind("IngressClass")),
					Name:  gatewayv1.ObjectName(ingressClass.Name),
				})
			}
		}
	}
	// delete AncestorRef which is not exist in the all parentRefs
	updateDeleteAncestors(r.Client, r.Log, list.Items, parentRefs)

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

func updateDeleteAncestors(client client.Client, logger logr.Logger, policies []v1alpha1.HTTPRoutePolicy, parentRefs []gatewayv1.ParentReference) {
	for i := range policies {
		policy := policies[i]
		length := len(policy.Status.Ancestors)
		policy.Status.Ancestors = slices.DeleteFunc(policy.Status.Ancestors, func(status v1alpha2.PolicyAncestorStatus) bool {
			return !slices.ContainsFunc(parentRefs, func(ref gatewayv1.ParentReference) bool {
				return parentRefValueEqual(status.AncestorRef, ref)
			})
		})
		if length != len(policy.Status.Ancestors) {
			if err := client.Status().Update(context.Background(), &policy); err != nil {
				logger.Error(err, "failed to update HTTPRoutePolicy status")
			}
		}
	}
}
