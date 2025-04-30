package controller

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/samber/lo"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

const (
	KindGateway      = "Gateway"
	KindHTTPRoute    = "HTTPRoute"
	KindGatewayClass = "GatewayClass"
	KindIngress      = "Ingress"
	KindIngressClass = "IngressClass"
	KindGatewayProxy = "GatewayProxy"
)

var (
	GatewaySecretChan = make(chan event.GenericEvent, 100)
)

const defaultIngressClassAnnotation = "ingressclass.kubernetes.io/is-default-class"

// IsDefaultIngressClass returns whether an IngressClass is the default IngressClass.
func IsDefaultIngressClass(obj client.Object) bool {
	if ingressClass, ok := obj.(*networkingv1.IngressClass); ok {
		return ingressClass.Annotations[defaultIngressClassAnnotation] == "true"
	}
	return false
}

func acceptedMessage(kind string) string {
	return fmt.Sprintf("the %s has been accepted by the api7-ingress-controller", kind)
}

func MergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	if newCondition.LastTransitionTime.IsZero() {
		newCondition.LastTransitionTime = metav1.Now()
	}
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
	gw.Status.Conditions = MergeCondition(gw.Status.Conditions, newCondition)
}

func setListenerCondition(gw *gatewayv1.Gateway, listenerName string, newCondition metav1.Condition) {
	for i, listener := range gw.Status.Listeners {
		if listener.Name == gatewayv1.SectionName(listenerName) {
			gw.Status.Listeners[i].Conditions = MergeCondition(listener.Conditions, newCondition)
			return
		}
	}
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

func SetGatewayConditionAccepted(gw *gatewayv1.Gateway, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.GatewayConditionAccepted),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.GatewayReasonAccepted),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setGatewayCondition(gw, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionAccepted(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionAccepted),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.ListenerConditionAccepted),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionProgrammed(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionProgrammed),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.ListenerReasonProgrammed),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionResolvedRefs(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionResolvedRefs),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.ListenerReasonResolvedRefs),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayConditionProgrammed(gw *gatewayv1.Gateway, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.GatewayConditionProgrammed),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.GatewayReasonProgrammed),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setGatewayCondition(gw, condition)
		ok = true
	}
	return
}

func SetRouteConditionAccepted(routeParentStatus *gatewayv1.RouteParentStatus, generation int64, status bool, message string) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionAccepted),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.RouteReasonAccepted),
		ObservedGeneration: generation,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(routeParentStatus.Conditions, condition) {
		routeParentStatus.Conditions = MergeCondition(routeParentStatus.Conditions, condition)
	}
}

func SetRouteConditionResolvedRefs(routeParentStatus *gatewayv1.RouteParentStatus, generation int64, status bool, message string) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionResolvedRefs),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.RouteReasonResolvedRefs),
		ObservedGeneration: generation,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(routeParentStatus.Conditions, condition) {
		routeParentStatus.Conditions = MergeCondition(routeParentStatus.Conditions, condition)
	}
}

func SetRouteParentRef(routeParentStatus *gatewayv1.RouteParentStatus, gatewayName string, namespace string) {
	kind := gatewayv1.Kind(KindGateway)
	group := gatewayv1.Group(gatewayv1.GroupName)
	ns := gatewayv1.Namespace(namespace)
	routeParentStatus.ParentRef = gatewayv1.ParentReference{
		Kind:      &kind,
		Group:     &group,
		Name:      gatewayv1.ObjectName(gatewayName),
		Namespace: &ns,
	}
	routeParentStatus.ControllerName = gatewayv1.GatewayController(config.ControllerConfig.ControllerName)
}

func ParseRouteParentRefs(
	ctx context.Context, mgrc client.Client, route client.Object, parentRefs []gatewayv1.ParentReference,
) ([]RouteParentRefContext, error) {
	gateways := make([]RouteParentRefContext, 0)
	for _, parentRef := range parentRefs {
		namespace := route.GetNamespace()
		if parentRef.Namespace != nil {
			namespace = string(*parentRef.Namespace)
		}
		name := string(parentRef.Name)

		if parentRef.Kind != nil && *parentRef.Kind != KindGateway {
			continue
		}

		gateway := gatewayv1.Gateway{}
		if err := mgrc.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &gateway); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return nil, fmt.Errorf("failed to retrieve gateway for route: %w", err)
		}

		gatewayClass := gatewayv1.GatewayClass{}
		if err := mgrc.Get(ctx, client.ObjectKey{
			Name: string(gateway.Spec.GatewayClassName),
		}, &gatewayClass); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return nil, fmt.Errorf("failed to retrieve gatewayclass for gateway: %w", err)
		}

		if string(gatewayClass.Spec.ControllerName) != config.ControllerConfig.ControllerName {
			continue
		}

		matched := false
		reason := gatewayv1.RouteReasonNoMatchingParent
		var listenerName string

		for _, listener := range gateway.Spec.Listeners {
			if parentRef.SectionName != nil {
				if *parentRef.SectionName != "" && *parentRef.SectionName != listener.Name {
					continue
				}
			}

			if parentRef.Port != nil {
				if *parentRef.Port != listener.Port {
					continue
				}
			}

			if !routeMatchesListenerType(route, listener) {
				continue
			}

			if !routeHostnamesIntersectsWithListenerHostname(route, listener) {
				reason = gatewayv1.RouteReasonNoMatchingListenerHostname
				continue
			}

			if ok, err := routeMatchesListenerAllowedRoutes(ctx, mgrc, route, listener.AllowedRoutes, gateway.Namespace, parentRef.Namespace); err != nil {
				return nil, fmt.Errorf("failed matching listener %s to a route %s for gateway %s: %w",
					listener.Name, route.GetName(), gateway.Name, err,
				)
			} else if !ok {
				reason = gatewayv1.RouteReasonNotAllowedByListeners
				continue
			}

			// TODO: check if the listener status is programmed

			matched = true
			listenerName = string(listener.Name)
			break
		}

		if matched {
			gateways = append(gateways, RouteParentRefContext{
				Gateway:      &gateway,
				ListenerName: listenerName,
				Conditions: []metav1.Condition{{
					Type:               string(gatewayv1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					Reason:             string(gatewayv1.RouteReasonAccepted),
					ObservedGeneration: route.GetGeneration(),
				}},
			})
		} else {
			gateways = append(gateways, RouteParentRefContext{
				Gateway: &gateway,
				Conditions: []metav1.Condition{{
					Type:               string(gatewayv1.RouteConditionAccepted),
					Status:             metav1.ConditionFalse,
					Reason:             string(reason),
					ObservedGeneration: route.GetGeneration(),
				}},
			})
		}
	}

	return gateways, nil
}

func checkRouteAcceptedByListener(
	ctx context.Context,
	mgrc client.Client,
	route client.Object,
	gateway gatewayv1.Gateway,
	listener gatewayv1.Listener,
	parentRef gatewayv1.ParentReference,
) (bool, gatewayv1.RouteConditionReason, error) {
	if parentRef.SectionName != nil {
		if *parentRef.SectionName != "" && *parentRef.SectionName != listener.Name {
			return false, gatewayv1.RouteReasonNoMatchingParent, nil
		}
	}
	if parentRef.Port != nil {
		if *parentRef.Port != listener.Port {
			return false, gatewayv1.RouteReasonNoMatchingParent, nil
		}
	}
	if !routeMatchesListenerType(route, listener) {
		return false, gatewayv1.RouteReasonNoMatchingParent, nil
	}
	if !routeHostnamesIntersectsWithListenerHostname(route, listener) {
		return false, gatewayv1.RouteReasonNoMatchingListenerHostname, nil
	}
	if ok, err := routeMatchesListenerAllowedRoutes(ctx, mgrc, route, listener.AllowedRoutes, gateway.Namespace, parentRef.Namespace); err != nil {
		return false, gatewayv1.RouteReasonNotAllowedByListeners, fmt.Errorf("failed matching listener %s to a route %s for gateway %s: %w",
			listener.Name, route.GetName(), gateway.Name, err,
		)
	} else if !ok {
		return false, gatewayv1.RouteReasonNotAllowedByListeners, nil
	}
	return true, gatewayv1.RouteReasonAccepted, nil
}

func routeHostnamesIntersectsWithListenerHostname(route client.Object, listener gatewayv1.Listener) bool {
	switch r := route.(type) {
	case *gatewayv1.HTTPRoute:
		return listenerHostnameIntersectWithRouteHostnames(listener, r.Spec.Hostnames)
	default:
		return false
	}
}

func listenerHostnameIntersectWithRouteHostnames(listener gatewayv1.Listener, hostnames []gatewayv1.Hostname) bool {
	if len(hostnames) == 0 {
		return true
	}

	// if the listener has no hostname, all hostnames automatically intersect
	if listener.Hostname == nil || *listener.Hostname == "" {
		return true
	}

	// iterate over all the hostnames and check that at least one intersect with the listener hostname
	for _, hostname := range hostnames {
		if HostnamesIntersect(string(*listener.Hostname), string(hostname)) {
			return true
		}
	}

	return false
}

func HostnamesIntersect(a, b string) bool {
	return HostnamesMatch(a, b) || HostnamesMatch(b, a)
}

func HostnamesMatch(hostnameA, hostnameB string) bool {
	labelsA := strings.Split(hostnameA, ".")
	labelsB := strings.Split(hostnameB, ".")

	var i, j int
	var wildcard bool

	for i, j = 0, 0; i < len(labelsA) && j < len(labelsB); i, j = i+1, j+1 {
		if wildcard {
			for ; j < len(labelsB); j++ {
				if labelsA[i] == labelsB[j] {
					break
				}
			}
			if j == len(labelsB) {
				return false
			}
		}

		if labelsA[i] == "*" {
			wildcard = true
			j--
			continue
		}

		wildcard = false

		if labelsA[i] != labelsB[j] {
			return false
		}
	}

	return len(labelsA)-i == len(labelsB)-j
}

func routeMatchesListenerAllowedRoutes(
	ctx context.Context,
	mgrc client.Client,
	route client.Object,
	allowedRoutes *gatewayv1.AllowedRoutes,
	gatewayNamespace string,
	parentRefNamespace *gatewayv1.Namespace,
) (bool, error) {
	if allowedRoutes == nil {
		return true, nil
	}

	if !isRouteKindAllowed(route, allowedRoutes.Kinds) {
		return false, fmt.Errorf("route %s/%s is not allowed in the kind", route.GetNamespace(), route.GetName())
	}

	if !isRouteNamespaceAllowed(ctx, route, mgrc, gatewayNamespace, parentRefNamespace, allowedRoutes.Namespaces) {
		return false, fmt.Errorf("route %s/%s is not allowed in the namespace", route.GetNamespace(), route.GetName())
	}

	return true, nil
}

func isRouteKindAllowed(route client.Object, kinds []gatewayv1.RouteGroupKind) (ok bool) {
	ok = true
	if len(kinds) > 0 {
		_, ok = lo.Find(kinds, func(rgk gatewayv1.RouteGroupKind) bool {
			gvk := route.GetObjectKind().GroupVersionKind()
			return (rgk.Group != nil && string(*rgk.Group) == gvk.Group) && string(rgk.Kind) == gvk.Kind
		})
	}
	return
}

func isRouteNamespaceAllowed(
	ctx context.Context,
	route client.Object,
	mgrc client.Client,
	gatewayNamespace string,
	parentRefNamespace *gatewayv1.Namespace,
	routeNamespaces *gatewayv1.RouteNamespaces,
) bool {
	if routeNamespaces == nil || routeNamespaces.From == nil {
		return true
	}

	switch *routeNamespaces.From {
	case gatewayv1.NamespacesFromAll:
		return true

	case gatewayv1.NamespacesFromSame:
		if parentRefNamespace == nil {
			return gatewayNamespace == route.GetNamespace()
		}
		return route.GetNamespace() == string(*parentRefNamespace)

	case gatewayv1.NamespacesFromSelector:
		namespace := corev1.Namespace{}
		if err := mgrc.Get(ctx, client.ObjectKey{Name: route.GetNamespace()}, &namespace); err != nil {
			return false
		}

		s, err := metav1.LabelSelectorAsSelector(routeNamespaces.Selector)
		if err != nil {
			return false
		}
		return s.Matches(labels.Set(namespace.Labels))
	default:
		return true
	}
}

func routeMatchesListenerType(route client.Object, listener gatewayv1.Listener) bool {
	switch route.(type) {
	case *gatewayv1.HTTPRoute:
		if listener.Protocol != gatewayv1.HTTPProtocolType && listener.Protocol != gatewayv1.HTTPSProtocolType {
			return false
		}

		if listener.Protocol == gatewayv1.HTTPSProtocolType {
			if listener.TLS == nil {
				return false
			}

			if listener.TLS.Mode != nil && *listener.TLS.Mode != gatewayv1.TLSModeTerminate {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func getAttachedRoutesForListener(ctx context.Context, mgrc client.Client, gateway gatewayv1.Gateway, listener gatewayv1.Listener) (int32, error) {
	httpRouteList := gatewayv1.HTTPRouteList{}
	if err := mgrc.List(ctx, &httpRouteList); err != nil {
		return 0, err
	}
	var attachedRoutes int32
	for _, route := range httpRouteList.Items {
		route := route
		acceptedByGateway := lo.ContainsBy(route.Status.Parents, func(parentStatus gatewayv1.RouteParentStatus) bool {
			parentRef := parentStatus.ParentRef
			if parentRef.Group != nil && *parentRef.Group != gatewayv1.GroupName {
				return false
			}
			if parentRef.Kind != nil && *parentRef.Kind != KindGateway {
				return false
			}
			gatewayNamespace := route.Namespace
			if parentRef.Namespace != nil {
				gatewayNamespace = string(*parentRef.Namespace)
			}
			return gateway.Namespace == gatewayNamespace && gateway.Name == string(parentRef.Name)
		})
		if !acceptedByGateway {
			continue
		}

		for _, parentRef := range route.Spec.ParentRefs {
			ok, _, err := checkRouteAcceptedByListener(
				ctx,
				mgrc,
				&route,
				gateway,
				listener,
				parentRef,
			)
			if err != nil {
				return 0, err
			}
			if ok {
				attachedRoutes++
			}
		}
	}
	return attachedRoutes, nil
}

func getListenerStatus(
	ctx context.Context,
	mrgc client.Client,
	gateway *gatewayv1.Gateway,
) ([]gatewayv1.ListenerStatus, error) {
	statuses := make(map[gatewayv1.SectionName]gatewayv1.ListenerStatus, len(gateway.Spec.Listeners))

	for i, listener := range gateway.Spec.Listeners {
		attachedRoutes, err := getAttachedRoutesForListener(ctx, mrgc, *gateway, listener)
		if err != nil {
			return nil, err
		}
		var (
			reasonResolvedRef = string(gatewayv1.ListenerReasonResolvedRefs)
			statusResolvedRef = metav1.ConditionTrue

			reasonProgrammed = string(gatewayv1.ListenerReasonProgrammed)
			statusProgrammed = metav1.ConditionTrue

			supportedKinds = []gatewayv1.RouteGroupKind{}
		)

		if listener.AllowedRoutes == nil || listener.AllowedRoutes.Kinds == nil {
			supportedKinds = []gatewayv1.RouteGroupKind{
				{
					Kind: gatewayv1.Kind("HTTPRoute"),
				},
			}
		} else {
			for _, kind := range listener.AllowedRoutes.Kinds {
				if kind.Group != nil && *kind.Group != gatewayv1.GroupName {
					reasonResolvedRef = string(gatewayv1.ListenerReasonInvalidRouteKinds)
					statusResolvedRef = metav1.ConditionFalse
					continue
				}
				switch kind.Kind {
				case gatewayv1.Kind("HTTPRoute"):
					supportedKinds = append(supportedKinds, kind)
				default:
					reasonResolvedRef = string(gatewayv1.ListenerReasonInvalidRouteKinds)
					statusResolvedRef = metav1.ConditionFalse
				}

			}
		}

		if listener.TLS != nil {
			// TODO: support TLS
			secret := corev1.Secret{}
			resolved := true
			for _, ref := range listener.TLS.CertificateRefs {
				ns := gateway.Namespace
				if ref.Namespace != nil {
					ns = string(*ref.Namespace)
				}
				if err := mrgc.Get(ctx, client.ObjectKey{
					Namespace: ns,
					Name:      string(ref.Name),
				}, &secret); err != nil {
					resolved = false
					break
				}
			}
			if !resolved {
				reasonResolvedRef = string(gatewayv1.ListenerReasonInvalidCertificateRef)
				statusResolvedRef = metav1.ConditionFalse
				statusProgrammed = metav1.ConditionFalse
			}
		}

		conditions := []metav1.Condition{
			{
				Type:               string(gatewayv1.ListenerConditionProgrammed),
				Status:             statusProgrammed,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Now(),
				Reason:             reasonProgrammed,
			},
			{
				Type:               string(gatewayv1.ListenerConditionAccepted),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Now(),
				Reason:             string(gatewayv1.ListenerReasonAccepted),
			},
			{
				Type:               string(gatewayv1.ListenerConditionConflicted),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Now(),
				Reason:             string(gatewayv1.ListenerReasonNoConflicts),
			},
			{
				Type:               string(gatewayv1.ListenerConditionResolvedRefs),
				Status:             statusResolvedRef,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Now(),
				Reason:             reasonResolvedRef,
			},
		}

		status := gatewayv1.ListenerStatus{
			Name:           listener.Name,
			Conditions:     conditions,
			SupportedKinds: supportedKinds,
			AttachedRoutes: attachedRoutes,
		}

		changed := false
		if len(gateway.Status.Listeners) > i {
			if gateway.Status.Listeners[i].AttachedRoutes != attachedRoutes {
				changed = true
			}
			for _, condition := range conditions {
				if !IsConditionPresentAndEqual(gateway.Status.Listeners[i].Conditions, condition) {
					changed = true
					break
				}
			}
		} else {
			changed = true
		}

		if changed {
			statuses[listener.Name] = status
		} else {
			statuses[listener.Name] = gateway.Status.Listeners[i]
		}
	}

	// check for conflicts

	statusArray := []gatewayv1.ListenerStatus{}
	for _, status := range statuses {
		statusArray = append(statusArray, status)
	}

	return statusArray, nil
}

// SplitMetaNamespaceKey returns the namespace and name that
// MetaNamespaceKeyFunc encoded into key.
func SplitMetaNamespaceKey(key string) (namespace, name string, err error) {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 1:
		// name only, no namespace
		return "", parts[0], nil
	case 2:
		// namespace and name
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected key format: %q", key)
}

func ProcessGatewayProxy(r client.Client, tctx *provider.TranslateContext, gateway *gatewayv1.Gateway, rk provider.ResourceKind) error {
	if gateway == nil {
		return nil
	}
	infra := gateway.Spec.Infrastructure
	if infra == nil || infra.ParametersRef == nil {
		return nil
	}

	gatewayKind := provider.ResourceKind{
		Kind:      gateway.Kind,
		Namespace: gateway.Namespace,
		Name:      gateway.Name,
	}

	ns := gateway.GetNamespace()
	paramRef := infra.ParametersRef
	if string(paramRef.Group) == v1alpha1.GroupVersion.Group && string(paramRef.Kind) == KindGatewayProxy {
		gatewayProxy := &v1alpha1.GatewayProxy{}
		if err := r.Get(context.Background(), client.ObjectKey{
			Namespace: ns,
			Name:      paramRef.Name,
		}, gatewayProxy); err != nil {
			log.Errorw("failed to get GatewayProxy", zap.String("namespace", ns), zap.String("name", paramRef.Name), zap.Error(err))
			return err
		} else {
			log.Infow("found GatewayProxy for Gateway", zap.String("namespace", gateway.Namespace), zap.String("name", gateway.Name))
			tctx.GatewayProxies[gatewayKind] = *gatewayProxy
			tctx.ResourceParentRefs[rk] = append(tctx.ResourceParentRefs[rk], gatewayKind)

			// Process provider secrets if provider exists
			if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane {
				if gatewayProxy.Spec.Provider.ControlPlane != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

					secretRef := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
					secret := &corev1.Secret{}
					if err := r.Get(context.Background(), client.ObjectKey{
						Namespace: ns,
						Name:      secretRef.Name,
					}, secret); err != nil {
						log.Error(err, "failed to get secret for GatewayProxy provider",
							"namespace", ns,
							"name", secretRef.Name)
						return err
					}

					log.Info("found secret for GatewayProxy provider",
						"gateway", gateway.Name,
						"gatewayproxy", gatewayProxy.Name,
						"secret", secretRef.Name)

					tctx.Secrets[types.NamespacedName{
						Namespace: ns,
						Name:      secretRef.Name,
					}] = secret
				}
			}
		}
	}

	_, ok := tctx.GatewayProxies[gatewayKind]
	if !ok {
		return fmt.Errorf("no gateway proxy found for gateway: %s", gateway.Name)
	}

	return nil
}

// FullTypeName returns the fully qualified name of the type of the given value.
func FullTypeName(a any) string {
	typeOf := reflect.TypeOf(a)
	pkgPath := typeOf.PkgPath()
	name := typeOf.String()
	if typeOf.Kind() == reflect.Ptr {
		pkgPath = typeOf.Elem().PkgPath()
	}
	return path.Join(path.Dir(pkgPath), name)
}
