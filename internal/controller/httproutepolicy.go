package controller

import (
	"context"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		checker = conflictChecker{
			object:   httpRoute,
			policies: make(map[targetRefKey][]v1alpha1.HTTPRoutePolicy),
		}
		listForAllRules v1alpha1.HTTPRoutePolicyList
		key             = indexer.GenHTTPRoutePolicyIndexKey(gatewayv1.GroupName, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName(), "")
	)
	if err := r.List(context.Background(), &listForAllRules, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	for _, item := range listForAllRules.Items {
		checker.append("", item)
		tctx.HTTPRoutePolicies["*"] = append(tctx.HTTPRoutePolicies["*"], item)
	}

	for _, rule := range httpRoute.Spec.Rules {
		if rule.Name == nil {
			continue
		}

		var (
			ruleName            = string(*rule.Name)
			listForSectionRules v1alpha1.HTTPRoutePolicyList
			key                 = indexer.GenHTTPRoutePolicyIndexKey(gatewayv1.GroupName, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName(), ruleName)
		)
		if err := r.List(context.Background(), &listForSectionRules, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
			continue
		}
		for _, item := range listForSectionRules.Items {
			checker.append(ruleName, item)
			tctx.HTTPRoutePolicies[ruleName] = append(tctx.HTTPRoutePolicies[ruleName], item)
		}
	}

	// todo: unreachable
	// if the HTTPRoute is deleted, clear tctx.HTTPRoutePolicies and delete Ancestors from HTTPRoutePolicies status
	// if !httpRoute.GetDeletionTimestamp().IsZero() {
	// 	for _, policies := range checker.policies {
	// 		for i := range policies {
	// 			policy := policies[i]
	// 			_ = DeleteAncestors(&policy.Status, httpRoute.Spec.ParentRefs)
	// 			data, _ := json.Marshal(policy.Status)
	// 			r.Log.Info("policy status after delete ancestor", "data", string(data))
	// 			if err := r.Status().Update(context.Background(), &policy); err != nil {
	// 				r.Log.Error(err, "failed to Update policy status")
	// 			}
	// 			// tctx.StatusUpdaters = append(tctx.StatusUpdaters, &policy)
	// 		}
	// 	}
	// 	return nil
	// }

	var (
		status  = true
		reason  = string(v1alpha2.PolicyReasonAccepted)
		message string
	)
	if checker.conflict {
		status = false
		reason = string(v1alpha2.PolicyReasonConflicted)
		message = "HTTPRoutePolicy conflict with others target to the HTTPRoute"

		// clear HTTPRoutePolices from TranslateContext
		tctx.HTTPRoutePolicies = make(map[string][]v1alpha1.HTTPRoutePolicy)
	}

	for _, policies := range checker.policies {
		for i := range policies {
			policy := policies[i]
			r.modifyHTTPRoutePolicyStatus(httpRoute, &policy, status, reason, message)
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, &policy)
		}
	}

	return nil
}

func (r *HTTPRouteReconciler) modifyHTTPRoutePolicyStatus(httpRoute *gatewayv1.HTTPRoute, policy *v1alpha1.HTTPRoutePolicy, status bool, reason, message string) {
	condition := metav1.Condition{
		Type:               string(v1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             reason,
		Message:            message,
	}
	if !status {
		condition.Status = metav1.ConditionFalse
	}
	_ = SetAncestors(&policy.Status, httpRoute.Spec.ParentRefs, condition)
}

func (r *IngressReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	var (
		checker = conflictChecker{
			object:   ingress,
			policies: make(map[targetRefKey][]v1alpha1.HTTPRoutePolicy),
			conflict: false,
		}
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenHTTPRoutePolicyIndexKey(networkingv1.GroupName, "Ingress", ingress.GetNamespace(), ingress.GetName(), "")
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	for _, item := range list.Items {
		checker.append("", item)
		tctx.HTTPRoutePolicies["*"] = append(tctx.HTTPRoutePolicies["*"], item)
	}

	if checker.conflict {
		// clear HTTPRoutePolicies from TranslateContext
		tctx.HTTPRoutePolicies = make(map[string][]v1alpha1.HTTPRoutePolicy)
	}

	// todo: handle HTTPRoutePolicy status

	return nil
}

type conflictChecker struct {
	object   client.Object
	policies map[targetRefKey][]v1alpha1.HTTPRoutePolicy
	conflict bool
}

type targetRefKey struct {
	Group       gatewayv1.Group
	Namespace   gatewayv1.Namespace
	Name        gatewayv1.ObjectName
	SectionName gatewayv1.SectionName
}

func (c *conflictChecker) append(sectionName string, policy v1alpha1.HTTPRoutePolicy) {
	key := targetRefKey{
		Group:       gatewayv1.GroupName,
		Namespace:   gatewayv1.Namespace(c.object.GetNamespace()),
		Name:        gatewayv1.ObjectName(c.object.GetName()),
		SectionName: gatewayv1.SectionName(sectionName),
	}
	c.policies[key] = append(c.policies[key], policy)

	if !c.conflict {
	Loop:
		for _, items := range c.policies {
			for _, item := range items {
				if !ptr.Equal(item.Spec.Priority, policy.Spec.Priority) {
					c.conflict = true
					break Loop
				}
			}
		}
	}
}
