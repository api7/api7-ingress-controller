package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type RouteParentRefContext struct {
	Gateway *gatewayv1.Gateway

	ListenerName string
	Conditions   []metav1.Condition
}
