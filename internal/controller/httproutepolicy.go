package controller

import (
	"context"
	"slices"
	"time"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (r *HTTPRouteReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	// list HTTPRoutePolices which sectionName is not specified
	var (
		checker = conflictChecker{
			httpRoute: httpRoute,
			policies:  make(map[ancestorRefKey][]v1alpha1.HTTPRoutePolicy),
		}
		listForAllRules v1alpha1.HTTPRoutePolicyList
		key             = indexer.GenHTTPRoutePolicyIndexKey(v1alpha1.GroupVersion.Group, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName(), "")
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
			key                 = indexer.GenHTTPRoutePolicyIndexKey(v1alpha1.GroupVersion.Group, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName(), ruleName)
		)
		if err := r.List(context.Background(), &listForSectionRules, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
			continue
		}
		for _, item := range listForSectionRules.Items {
			checker.append(ruleName, item)
			tctx.HTTPRoutePolicies[ruleName] = append(tctx.HTTPRoutePolicies[ruleName], item)
		}
	}

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

	for key, policies := range checker.policies {
		for _, policy := range policies {
			r.modifyHTTPRoutePolicyStatus(key, &policy, status, reason, message)
			if err := r.Status().Update(context.Background(), &policy); err != nil {
				r.Log.Error(err, "failed to update HTTPRoutePolicyStatus")
			}
		}
	}

	return nil
}

func (r *HTTPRouteReconciler) clearHTTPRoutePolicyRedundantAncestor(policy *v1alpha1.HTTPRoutePolicy) {
	var keys = make(map[ancestorRefKey]struct{})
	for _, ref := range policy.Spec.TargetRefs {
		key := ancestorRefKey{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Namespace: gatewayv1.Namespace(policy.GetNamespace()),
			Name:      ref.Name,
		}
		if ref.SectionName != nil {
			key.SectionName = *ref.SectionName
		}
		r.Log.Info("clearHTTPRoutePolicyRedundantAncestor", "keys[]", key)
		keys[key] = struct{}{}
	}

	policy.Status.Ancestors = slices.DeleteFunc(policy.Status.Ancestors, func(ancestor v1alpha2.PolicyAncestorStatus) bool {
		key := ancestorRefKey{
			Namespace: gatewayv1.Namespace(policy.GetNamespace()),
			Name:      ancestor.AncestorRef.Name,
		}
		if ancestor.AncestorRef.Group != nil {
			key.Group = *ancestor.AncestorRef.Group
		}
		if ancestor.AncestorRef.Kind != nil {
			key.Kind = *ancestor.AncestorRef.Kind
		}
		if ancestor.AncestorRef.SectionName != nil {
			key.SectionName = *ancestor.AncestorRef.SectionName
		}
		r.Log.Info("clearHTTPRoutePolicyRedundantAncestor", "key", key)
		_, ok := keys[key]
		return !ok
	})
}

func (r *HTTPRouteReconciler) modifyHTTPRoutePolicyStatus(key ancestorRefKey, policy *v1alpha1.HTTPRoutePolicy, status bool, reason, message string) {
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
	var hasAncestor bool
	for i, ancestor := range policy.Status.Ancestors {
		if ancestor.AncestorRef.Kind != nil && *ancestor.AncestorRef.Kind == key.Kind && ancestor.AncestorRef.Name == key.Name {
			ancestor.ControllerName = v1alpha2.GatewayController(config.GetControllerName())
			ancestor.Conditions = []metav1.Condition{condition}
			hasAncestor = true
		}
		policy.Status.Ancestors[i] = ancestor
	}
	if !hasAncestor {
		ref := v1alpha2.ParentReference{
			Group:     &key.Group,
			Kind:      &key.Kind,
			Namespace: &key.Namespace,
			Name:      key.Name,
		}
		if key.SectionName != "" {
			ref.SectionName = &key.SectionName
		}
		policy.Status.Ancestors = append(policy.Status.Ancestors, v1alpha2.PolicyAncestorStatus{
			AncestorRef:    ref,
			ControllerName: v1alpha2.GatewayController(config.GetControllerName()),
			Conditions:     []metav1.Condition{condition},
		})
	}
}

type conflictChecker struct {
	httpRoute *gatewayv1.HTTPRoute
	policies  map[ancestorRefKey][]v1alpha1.HTTPRoutePolicy
	conflict  bool
}

type ancestorRefKey struct {
	Group       gatewayv1.Group
	Kind        gatewayv1.Kind
	Namespace   gatewayv1.Namespace
	Name        gatewayv1.ObjectName
	SectionName gatewayv1.SectionName
}

func (c *conflictChecker) append(sectionName string, policy v1alpha1.HTTPRoutePolicy) {
	key := ancestorRefKey{
		Group:       gatewayv1.GroupName,
		Kind:        "HTTPRoute",
		Namespace:   gatewayv1.Namespace(c.httpRoute.GetNamespace()),
		Name:        gatewayv1.ObjectName(c.httpRoute.GetName()),
		SectionName: gatewayv1.SectionName(sectionName),
	}
	c.policies[key] = append(c.policies[key], policy)

	if !c.conflict {
	Loop:
		for _, items := range c.policies {
			for _, item := range items {
				if item.Spec.Priority != policy.Spec.Priority || *item.Spec.Priority != *policy.Spec.Priority {
					c.conflict = true
					break Loop
				}
			}
		}
	}
}
