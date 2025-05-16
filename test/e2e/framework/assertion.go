package framework

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func GatewayClassMustHaveCondition(t testing.TestingT, cli client.Client, timeout time.Duration, gcNN types.NamespacedName, condition metav1.Condition) {
	err := PollUntilGatewayClassMustHaveStatus(cli, timeout, gcNN, func(gc gatewayv1.GatewayClass) bool {
		if err := kubernetes.ConditionsHaveLatestObservedGeneration(&gc, gc.Status.Conditions); err != nil {
			log.Printf("GatewayClass %s %v", gcNN, err)
			return false
		}
		if findConditionInList(gc.Status.Conditions, condition) {
			return true
		}
		log.Printf("NOT FOUND condition %v in %v", condition, gc.Status.Conditions)
		return false
	})
	require.NoError(t, err, "waiting for GatewayClass to have condition %+v", condition)
}

func PollUntilGatewayClassMustHaveStatus(cli client.Client, timeout time.Duration, gcNN types.NamespacedName, f func(gc gatewayv1.GatewayClass) bool) error {
	if err := gatewayv1.Install(cli.Scheme()); err != nil {
		return err
	}
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		var gc gatewayv1.GatewayClass
		if err := cli.Get(ctx, gcNN, &gc); err != nil {
			return false, errors.Wrapf(err, "failed to get GatewayClass %s", gcNN)
		}
		return f(gc), nil
	})
}

func GatewayMustHaveCondition(t testing.TestingT, cli client.Client, timeout time.Duration, gwNN types.NamespacedName, condition metav1.Condition) {
	err := PollUntilGatewayHaveStatus(cli, timeout, gwNN, func(gw gatewayv1.Gateway) bool {
		if err := kubernetes.ConditionsHaveLatestObservedGeneration(&gw, gw.Status.Conditions); err != nil {
			log.Printf("Gateway %s %v", gwNN, err)
			return false
		}
		if findConditionInList(gw.Status.Conditions, condition) {
			log.Printf("found condition %v in list [%v]", condition, gw.Status.Conditions)
			return true
		} else {
			log.Printf("not found condition %v in list [%v]", condition, gw.Status.Conditions)
			return false
		}
	})
	require.NoError(t, err, "waiting for Gateway to have condition %+v", condition)
}

func PollUntilGatewayHaveStatus(cli client.Client, timeout time.Duration, gwNN types.NamespacedName, f func(gateway gatewayv1.Gateway) bool) error {
	if err := gatewayv1.Install(cli.Scheme()); err != nil {
		return err
	}
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		var gw gatewayv1.Gateway
		if err := cli.Get(ctx, gwNN, &gw); err != nil {
			return false, errors.Wrapf(err, "failed to get Gateway %s", gwNN)
		}
		return f(gw), nil
	})
}

func HTTPRouteMustHaveCondition(t testing.TestingT, cli client.Client, timeout time.Duration, refNN, hrNN types.NamespacedName, condition metav1.Condition) {
	err := PollUntilHTTPRouteHaveStatus(cli, timeout, hrNN, func(hr gatewayv1.HTTPRoute) bool {
		for _, parent := range hr.Status.Parents {
			if err := kubernetes.ConditionsHaveLatestObservedGeneration(&hr, parent.Conditions); err != nil {
				log.Printf("HTTPRoute %s (parentRef=%v) %v", hrNN, parentRefToString(parent.ParentRef), err)
				return false
			}
			if (refNN.Name == "" || parent.ParentRef.Name == gatewayv1.ObjectName(refNN.Name)) &&
				(refNN.Namespace == "" || (parent.ParentRef.Namespace != nil && string(*parent.ParentRef.Namespace) == refNN.Namespace)) {
				if findConditionInList(parent.Conditions, condition) {
					log.Printf("found condition %v in list [%v] for %s reference %s", condition, parent.Conditions, hrNN, refNN)
					return true
				} else {
					log.Printf("found condition %v in list [%v] for %s reference %s", condition, parent.Conditions, hrNN, refNN)
				}
			}
		}
		return false
	})
	require.NoError(t, err, "error waiting for HTTPRoute status to have a Condition matching %+v", condition)
}

func PollUntilHTTPRouteHaveStatus(cli client.Client, timeout time.Duration, hrNN types.NamespacedName, f func(route gatewayv1.HTTPRoute) bool) error {
	if err := gatewayv1.Install(cli.Scheme()); err != nil {
		return err
	}
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		var httpRoute gatewayv1.HTTPRoute
		if err := cli.Get(ctx, hrNN, &httpRoute); err != nil {
			return false, errors.Wrapf(err, "failed to get HTTPRoute %s", hrNN)
		}
		return f(httpRoute), nil
	})
}

func HTTPRoutePolicyMustHaveCondition(t testing.TestingT, client client.Client, timeout time.Duration, refNN, hrpNN types.NamespacedName,
	condition metav1.Condition) {
	err := PollUntilHTTPRoutePolicyHaveStatus(client, timeout, hrpNN, func(httpRoutePolicy v1alpha1.HTTPRoutePolicy, status v1alpha1.PolicyStatus) bool {
		for _, ancestor := range status.Ancestors {
			if err := kubernetes.ConditionsHaveLatestObservedGeneration(&httpRoutePolicy, ancestor.Conditions); err != nil {
				log.Printf("HTTPRoutePolicy %s (parentRef=%v) %v", hrpNN, parentRefToString(ancestor.AncestorRef), err)
				return false
			}

			if ancestor.AncestorRef.Name == gatewayv1.ObjectName(refNN.Name) &&
				(refNN.Namespace == "" || (ancestor.AncestorRef.Namespace != nil && string(*ancestor.AncestorRef.Namespace) == refNN.Namespace)) {
				if findConditionInList(ancestor.Conditions, condition) {
					log.Printf("found condition %v in list [%v] for %s reference %s", condition, ancestor.Conditions, hrpNN, refNN)
					return true
				} else {
					log.Printf("not found condition %v in list [%v] for %s reference %s", condition, ancestor.Conditions, hrpNN, refNN)
				}
			}
		}
		return false
	})

	require.NoError(t, err, "error waiting for HTTPRoutePolicy status to have a Condition matching %+v", condition)
}

func PollUntilHTTPRoutePolicyHaveStatus(client client.Client, timeout time.Duration, hrpNN types.NamespacedName,
	f func(httpRoutePolicy v1alpha1.HTTPRoutePolicy, status v1alpha1.PolicyStatus) bool) error {
	_ = v1alpha1.AddToScheme(client.Scheme())
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		var httpRoutePolicy v1alpha1.HTTPRoutePolicy
		if err = client.Get(ctx, hrpNN, &httpRoutePolicy); err != nil {
			return false, errors.Wrapf(err, "error fetching HTTPRoutePolicy %s", hrpNN)
		}
		return f(httpRoutePolicy, httpRoutePolicy.Status), nil
	})
}

func parentRefToString(p gatewayv1.ParentReference) string {
	if p.Namespace != nil && *p.Namespace != "" {
		return fmt.Sprintf("%v/%v", p.Namespace, p.Name)
	}
	return string(p.Name)
}

func findConditionInList(conditions []metav1.Condition, expected metav1.Condition) bool {
	return slices.ContainsFunc(conditions, func(item metav1.Condition) bool {
		// an empty Status string means "Match any status".
		// an empty Reason string means "Match any reason".
		if expected.Type == item.Type &&
			(expected.Status == "" || expected.Status == item.Status) &&
			(expected.Reason == "" || expected.Reason == item.Reason) &&
			expected.Message != "" && !strings.Contains(item.Message, expected.Message) {
			log.Printf("condition message not match, item.Message: %s, expected.Message: %s", item.Message, expected.Message)
		}
		return expected.Type == item.Type &&
			(expected.Status == "" || expected.Status == item.Status) &&
			(expected.Reason == "" || expected.Reason == item.Reason) &&
			(expected.Message == "" || strings.Contains(item.Message, expected.Message))
	})
}
