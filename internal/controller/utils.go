package controller

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"
)

func setGatewayCondition(gw *gatewayapi.Gateway, newCondition meta.Condition) {
	newConditions := []meta.Condition{}
	for _, condition := range gw.Status.Conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	gw.Status.Conditions = newConditions
}

func reconcileGatewaysMatchGatewayClass(gatewayClass client.Object, gateways []gatewayapi.Gateway) (recs []reconcile.Request) {
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

func IsConditionPresentAndEqual(conditions []meta.Condition, condition meta.Condition) bool {
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
