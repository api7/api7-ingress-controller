package controller

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func acceptedMessage(kind string) string {
	return fmt.Sprintf("the %s has been accepted by the api7-ingress-controller", kind)
}

func mergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	newConditions := []metav1.Condition{}
	for _, condition := range conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	return newConditions
}

func setGatewayCondition(gw *gatewayv1.Gateway, newCondition metav1.Condition) {
	gw.Status.Conditions = mergeCondition(gw.Status.Conditions, newCondition)
}

func reconcileGatewaysMatchGatewayClass(gatewayClass client.Object, gateways []gatewayv1.Gateway) (recs []reconcile.Request) {
	for _, gateway := range gateways {
		if string(gateway.Spec.GatewayClassName) == gatewayClass.GetName() {
			recs = append(recs, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      gateway.GetName(),
					Namespace: gateway.GetNamespace(),
				},
			})
		}
	}
	return
}

func IsConditionPresentAndEqual(conditions []metav1.Condition, condition metav1.Condition) bool {
	for _, cond := range conditions {
		if cond.Type == condition.Type &&
			cond.Reason == condition.Reason &&
			cond.Status == condition.Status &&
			cond.ObservedGeneration == condition.ObservedGeneration {
			return true
		}
	}
	return false
}

func SetRouteConditionAccepted(hr *gatewayv1.HTTPRoute, gatewayIdx int, status bool, message string) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionAccepted),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.RouteReasonAccepted),
		ObservedGeneration: hr.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(hr.Status.Parents[gatewayIdx].Conditions, condition) {
		hr.Status.Parents[gatewayIdx].Conditions = mergeCondition(hr.Status.Parents[0].Conditions, condition)
	}
}

func SetRouteConditionResolvedRefs(hr *gatewayv1.HTTPRoute, gatewayIdx int, status bool, message string) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionResolvedRefs),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.RouteReasonResolvedRefs),
		ObservedGeneration: hr.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(hr.Status.Parents[gatewayIdx].Conditions, condition) {
		hr.Status.Parents[gatewayIdx].Conditions = mergeCondition(hr.Status.Parents[0].Conditions, condition)
	}
}

func SetRouteStatusParentRef(hr *gatewayv1.HTTPRoute, gatewayIdx int, GatewayName string) {
	kind := gatewayv1.Kind("Gateway")
	group := gatewayv1.Group("gateway.networking.k8s.io")
	hr.Status.Parents[gatewayIdx].ParentRef = gatewayv1.ParentReference{
		Kind:  &kind,
		Group: &group,
		Name:  gatewayv1.ObjectName(GatewayName),
	}
}
