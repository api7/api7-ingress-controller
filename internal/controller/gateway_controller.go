package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update

// GatewayReconciler reconciles a Gateway object.
type GatewayReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
}

func (r *GatewayReconciler) setupIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.TODO(),
		&gatewayv1.Gateway{},
		indexer.ParametersRef,
		indexer.GatewayParametersRefIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.setupIndexer(mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}).
		Watches(
			&gatewayv1.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayForGatewayClass),
			builder.WithPredicates(predicate.NewPredicateFuncs(r.matchesGatewayClass)),
		).
		Watches(
			&gatewayv1.HTTPRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForHTTPRoute),
		).
		Watches(
			&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForGatewayProxy),
		).
		Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := new(gatewayv1.Gateway)
	if err := r.Get(ctx, req.NamespacedName, gateway); err != nil {
		if client.IgnoreNotFound(err) == nil {
			gateway.Namespace = req.Namespace
			gateway.Name = req.Name

			if err := r.Provider.Delete(ctx, gateway); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}
	ns := gateway.GetNamespace()
	if !r.checkGatewayClass(gateway) {
		return ctrl.Result{}, nil
	}

	conditionProgrammedStatus, conditionProgrammedMsg := true, "Programmed"

	cfg := config.GetFirstGatewayConfig()

	var addrs []gatewayv1.GatewayStatusAddress

	if len(gateway.Status.Addresses) != len(cfg.Addresses) {
		for _, addr := range cfg.Addresses {
			if addr == "" {
				continue
			}
			addrs = append(addrs,
				gatewayv1.GatewayStatusAddress{
					Value: addr,
				},
			)
		}
	}

	r.Log.Info("gateway has been accepted", "gateway", gateway.GetName())
	type status struct {
		status bool
		msg    string
	}
	acceptStatus := status{
		status: true,
		msg:    acceptedMessage("gateway"),
	}
	tctx := &provider.TranslateContext{
		Secrets: make(map[types.NamespacedName]*corev1.Secret),
	}
	r.processListenerConfig(tctx, gateway, ns)
	r.processGatewayProxy(tctx, gateway, ns)

	if err := r.Provider.Update(ctx, tctx, gateway); err != nil {
		acceptStatus = status{
			status: false,
			msg:    err.Error(),
		}
	}

	ListenerStatuses, err := getListenerStatus(ctx, r.Client, gateway)
	if err != nil {
		return ctrl.Result{}, err
	}

	accepted := SetGatewayConditionAccepted(gateway, acceptStatus.status, acceptStatus.msg)
	Programmed := SetGatewayConditionProgrammed(gateway, conditionProgrammedStatus, conditionProgrammedMsg)
	if accepted || Programmed || len(addrs) > 0 || len(ListenerStatuses) > 0 {
		if len(addrs) > 0 {
			gateway.Status.Addresses = addrs
		}
		if len(ListenerStatuses) > 0 {
			gateway.Status.Listeners = ListenerStatuses
		}

		return ctrl.Result{}, r.Status().Update(ctx, gateway)
	}
	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) matchesGatewayClass(obj client.Object) bool {
	gateway, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}
	return matchesController(string(gateway.Spec.ControllerName))
}

/*
	func (r *GatewayReconciler) matchesGatewayForControlPlaneConfig(obj client.Object) bool {
		gateway, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
			return false
		}
		cfg := config.GetControlPlaneConfigByGatewatName(gateway.GetName())
		ok = true
		if cfg == nil {
			ok = false
		}
		return ok
	}
*/

func (r *GatewayReconciler) listGatewayForGatewayClass(ctx context.Context, gatewayClass client.Object) []reconcile.Request {
	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(context.Background(), gatewayList); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway class",
			"gatewayclass", gatewayClass.GetName(),
		)
		return nil
	}

	/*
		gateways := []gatewayv1.Gateway{}
		for _, gateway := range gatewayList.Items {
			if cp := config.GetControlPlaneConfigByGatewatName(gateway.GetName()); cp != nil {
				gateways = append(gateways, gateway)
			}
		}
	*/
	return reconcileGatewaysMatchGatewayClass(gatewayClass, gatewayList.Items)
}

func (r *GatewayReconciler) checkGatewayClass(gateway *gatewayv1.Gateway) bool {
	gatewayClass := &gatewayv1.GatewayClass{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		r.Log.Error(err, "failed to get gateway class", "gateway", gateway.GetName(), "gatewayclass", gateway.Spec.GatewayClassName)
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}

func (r *GatewayReconciler) listGatewaysForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayProxy")
		return nil
	}
	namespace := gatewayProxy.GetNamespace()
	name := gatewayProxy.GetName()

	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(ctx, gatewayList, client.MatchingFields{
		indexer.ParametersRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	recs := make([]reconcile.Request, 0, len(gatewayList.Items))
	for _, gateway := range gatewayList.Items {
		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gateway.GetNamespace(),
				Name:      gateway.GetName(),
			},
		})
	}
	return recs
}

func (r *GatewayReconciler) listGatewaysForHTTPRoute(_ context.Context, obj client.Object) []reconcile.Request {
	httpRoute, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		r.Log.Error(
			fmt.Errorf("unexpected object type"),
			"HTTPRoute watch predicate received unexpected object type",
			"expected", "*gatewayapi.HTTPRoute", "found", reflect.TypeOf(obj),
		)
		return nil
	}
	recs := []reconcile.Request{}
	for _, routeParentStatus := range httpRoute.Status.Parents {
		gatewayNamespace := httpRoute.GetNamespace()
		parentRef := routeParentStatus.ParentRef
		if parentRef.Group != nil && *parentRef.Group != gatewayv1.GroupName {
			continue
		}
		if parentRef.Kind != nil && *parentRef.Kind != "Gateway" {
			continue
		}
		if parentRef.Namespace != nil {
			gatewayNamespace = string(*parentRef.Namespace)
		}

		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gatewayNamespace,
				Name:      string(parentRef.Name),
			},
		})
	}
	return recs
}

func (r *GatewayReconciler) processGatewayProxy(tctx *provider.TranslateContext, gateway *gatewayv1.Gateway, ns string) {
	infra := gateway.Spec.Infrastructure
	if infra != nil && infra.ParametersRef != nil {
		paramRef := infra.ParametersRef
		if string(paramRef.Group) == v1alpha1.GroupVersion.Group && string(paramRef.Kind) == "GatewayProxy" {
			gatewayProxy := &v1alpha1.GatewayProxy{}
			if err := r.Get(context.Background(), client.ObjectKey{
				Namespace: ns,
				Name:      string(paramRef.Name),
			}, gatewayProxy); err != nil {
				log.Error(err, "failed to get GatewayProxy", "namespace", ns, "name", string(paramRef.Name))
			} else {
				log.Info("found GatewayProxy for Gateway", "gateway", gateway.Name, "gatewayproxy", gatewayProxy.Name)
				tctx.GatewayProxy = gatewayProxy
			}
		}
	}
}

func (r *GatewayReconciler) processListenerConfig(tctx *provider.TranslateContext, gateway *gatewayv1.Gateway, ns string) {
	listeners := gateway.Spec.Listeners
	for _, listener := range listeners {
		if listener.TLS == nil || listener.TLS.CertificateRefs == nil {
			continue
		}
		secret := corev1.Secret{}
		for _, ref := range listener.TLS.CertificateRefs {
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			if ref.Kind != nil && *ref.Kind == gatewayv1.Kind("Secret") {
				if err := r.Get(context.Background(), client.ObjectKey{
					Namespace: ns,
					Name:      string(ref.Name),
				}, &secret); err != nil {
					log.Error(err, "failed to get secret", "namespace", ns, "name", string(ref.Name))
					SetGatewayListenerConditionProgrammed(gateway, string(listener.Name), false, err.Error())
					SetGatewayListenerConditionResolvedRefs(gateway, string(listener.Name), false, err.Error())
					break
				}
				log.Info("Setting secret for listener", "listener", listener.Name, "secret", secret.Name, " namespace", ns)
				tctx.Secrets[types.NamespacedName{Namespace: ns, Name: string(ref.Name)}] = &secret
			}
		}
	}
}
