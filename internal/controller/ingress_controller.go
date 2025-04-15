package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IngressReconciler reconciles a Ingress object.
type IngressReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		).
		Watches(
			&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesByService),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesBySecret),
		).
		Complete(r)
}

// Reconcile handles the reconciliation of Ingress resources
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ingress := new(networkingv1.Ingress)
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Ingress was deleted, clean up corresponding resources
			ingress.Namespace = req.Namespace
			ingress.Name = req.Name

			ingress.TypeMeta = metav1.TypeMeta{
				Kind:       KindIngress,
				APIVersion: networkingv1.SchemeGroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, ingress); err != nil {
				r.Log.Error(err, "failed to delete ingress resources", "ingress", ingress.Name)
				return ctrl.Result{}, err
			}
			r.Log.Info("deleted ingress resources", "ingress", ingress.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reconciling ingress", "ingress", ingress.Name)

	// create a translate context
	tctx := provider.NewDefaultTranslateContext()

	// process IngressClass parameters if they reference GatewayProxy
	if err := r.processIngressClassParameters(ctx, tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// process TLS configuration
	if err := r.processTLS(ctx, tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process TLS configuration", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// process backend services
	if err := r.processBackends(ctx, tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process backend services", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// update the ingress resources
	if err := r.Provider.Update(ctx, tctx, ingress); err != nil {
		r.Log.Error(err, "failed to update ingress resources", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// update the ingress status
	if err := r.updateStatus(ctx, ingress); err != nil {
		r.Log.Error(err, "failed to update ingress status", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getIngressClass get the ingress class for the ingress
func (r *IngressReconciler) getIngressClass(obj client.Object) (*networkingv1.IngressClass, error) {
	ingress := obj.(*networkingv1.Ingress)

	if ingress.Spec.IngressClassName == nil {
		// handle the case where IngressClassName is not specified
		// find all ingress classes and check if any of them is marked as default
		ingressClassList := &networkingv1.IngressClassList{}
		if err := r.List(context.Background(), ingressClassList, client.MatchingFields{
			indexer.IngressClass: config.GetControllerName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingress classes")
			return nil, err
		}

		// find the ingress class that is marked as default
		for _, ic := range ingressClassList.Items {
			if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
				log.Debugw("match the default ingress class")
				return &ic, nil
			}
		}

		log.Debugw("no default ingress class found")
		return nil, errors.New("no default ingress class found")
	}

	// if it does not match, check if the ingress class is controlled by us
	ingressClass := networkingv1.IngressClass{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: *ingress.Spec.IngressClassName}, &ingressClass); err != nil {
		return nil, err
	}

	if matchesController(ingressClass.Spec.Controller) {
		return &ingressClass, nil
	}

	return nil, errors.New("ingress class is not controlled by us")
}

// checkIngressClass check if the ingress uses the ingress class that we control
func (r *IngressReconciler) checkIngressClass(obj client.Object) bool {
	ingress := obj.(*networkingv1.Ingress)

	if ingress.Spec.IngressClassName == nil {
		// handle the case where IngressClassName is not specified
		// find all ingress classes and check if any of them is marked as default
		ingressClassList := &networkingv1.IngressClassList{}
		if err := r.List(context.Background(), ingressClassList, client.MatchingFields{
			indexer.IngressClass: config.GetControllerName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingress classes")
			return false
		}

		// find the ingress class that is marked as default
		for _, ic := range ingressClassList.Items {
			if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
				log.Debugw("match the default ingress class")
				return true
			}
		}

		log.Debugw("no default ingress class found")
		return false
	}

	configuredClass := config.GetIngressClass()
	// if the ingress class name matches the configured ingress class name, return true
	if *ingress.Spec.IngressClassName == configuredClass {
		log.Debugw("match the configured ingress class name")
		return true
	}

	// if it does not match, check if the ingress class is controlled by us
	ingressClass := networkingv1.IngressClass{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: *ingress.Spec.IngressClassName}, &ingressClass); err != nil {
		return false
	}

	return matchesController(ingressClass.Spec.Controller)
}

// matchesIngressController check if the ingress class is controlled by us
func (r *IngressReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return false
	}

	return matchesController(ingressClass.Spec.Controller)
}

// listIngressForIngressClass list all ingresses that use a specific ingress class
func (r *IngressReconciler) listIngressForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return nil
	}

	// check if the ingress class is the default ingress class
	if IsDefaultIngressClass(ingressClass) {
		ingressList := &networkingv1.IngressList{}
		if err := r.List(ctx, ingressList); err != nil {
			r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
			return nil
		}

		requests := make([]reconcile.Request, 0, len(ingressList.Items))
		for _, ingress := range ingressList.Items {
			if ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName == "" ||
				*ingress.Spec.IngressClassName == ingressClass.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: ingress.Namespace,
						Name:      ingress.Name,
					},
				})
			}
		}
		return requests
	} else {
		ingressList := &networkingv1.IngressList{}
		if err := r.List(ctx, ingressList, client.MatchingFields{
			indexer.IngressClassRef: ingressClass.GetName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
			return nil
		}

		requests := make([]reconcile.Request, 0, len(ingressList.Items))
		for _, ingress := range ingressList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}

		return requests
	}
}

// listIngressesByService list all ingresses that use a specific service
func (r *IngressReconciler) listIngressesByService(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}

	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by service", "service", serviceName)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if r.checkIngressClass(&ingress) {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}
	}
	return requests
}

// listIngressesBySecret list all ingresses that use a specific secret
func (r *IngressReconciler) listIngressesBySecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Secret")
		return nil
	}

	namespace := secret.GetNamespace()
	name := secret.GetName()

	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by secret", "secret", name)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if r.checkIngressClass(&ingress) {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}
	}
	return requests
}

// processTLS process the TLS configuration of the ingress
func (r *IngressReconciler) processTLS(ctx context.Context, tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		secret := corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: ingress.Namespace,
			Name:      tls.SecretName,
		}, &secret); err != nil {
			log.Error(err, "failed to get secret", "namespace", ingress.Namespace, "name", tls.SecretName)
			return err
		}

		if secret.Data == nil {
			log.Warnw("secret data is nil", zap.String("secret", secret.Namespace+"/"+secret.Name))
			continue
		}

		// add the secret to the translate context
		tctx.Secrets[types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}] = &secret
	}

	return nil
}

// processBackends process the backend services of the ingress
func (r *IngressReconciler) processBackends(ctx context.Context, tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	var terr error

	// process all the backend services in the rules
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			service := path.Backend.Service
			if err := r.processBackendService(ctx, tctx, ingress.Namespace, service); err != nil {
				terr = err
			}
		}
	}
	return terr
}

// processBackendService process a single backend service
func (r *IngressReconciler) processBackendService(ctx context.Context, tctx *provider.TranslateContext, namespace string, backendService *networkingv1.IngressServiceBackend) error {
	// get the service
	var service corev1.Service
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      backendService.Name,
	}, &service); err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.Log.Info("service not found", "namespace", namespace, "name", backendService.Name)
			return nil
		}
		return err
	}

	// verify if the port exists
	var portExists bool
	if backendService.Port.Number != 0 {
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Port == backendService.Port.Number {
				portExists = true
				break
			}
		}
	} else if backendService.Port.Name != "" {
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Name == backendService.Port.Name {
				portExists = true
				break
			}
		}
	}

	if !portExists {
		err := fmt.Errorf("port(name: %s, number: %d) not found in service %s/%s", backendService.Port.Name, backendService.Port.Number, namespace, backendService.Name)
		r.Log.Error(err, "service port not found")
		return err
	}

	// get the endpoint slices
	endpointSliceList := &discoveryv1.EndpointSliceList{}
	if err := r.List(ctx, endpointSliceList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			discoveryv1.LabelServiceName: backendService.Name,
		},
	); err != nil {
		r.Log.Error(err, "failed to list endpoint slices", "namespace", namespace, "name", backendService.Name)
		return err
	}

	// save the endpoint slices to the translate context
	tctx.EndpointSlices[client.ObjectKey{
		Namespace: namespace,
		Name:      backendService.Name,
	}] = endpointSliceList.Items

	tctx.Services[client.ObjectKey{
		Namespace: namespace,
		Name:      backendService.Name,
	}] = &service

	return nil
}

// updateStatus update the status of the ingress
func (r *IngressReconciler) updateStatus(ctx context.Context, ingress *networkingv1.Ingress) error {
	var loadBalancerStatus networkingv1.IngressLoadBalancerStatus

	// todo: remove using default config, use the StatusAddress And PublishService in the gateway proxy

	// 1. use the IngressStatusAddress in the config
	statusAddresses := config.GetIngressStatusAddress()
	if len(statusAddresses) > 0 {
		for _, addr := range statusAddresses {
			if addr == "" {
				continue
			}
			loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
				IP: addr,
			})
		}
	} else {
		// 2. if the IngressStatusAddress is not configured, try to use the PublishService
		publishService := config.GetIngressPublishService()
		if publishService != "" {
			// parse the namespace/name format
			namespace, name, err := SplitMetaNamespaceKey(publishService)
			if err != nil {
				return fmt.Errorf("invalid ingress-publish-service format: %s, expected format: namespace/name", publishService)
			}
			// if the namespace is not specified, use the ingress namespace
			if namespace == "" {
				namespace = ingress.Namespace
			}

			svc := &corev1.Service{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc); err != nil {
				return fmt.Errorf("failed to get publish service %s: %w", publishService, err)
			}

			if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
				// get the LoadBalancer IP and Hostname of the service
				for _, ip := range svc.Status.LoadBalancer.Ingress {
					if ip.IP != "" {
						loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
							IP: ip.IP,
						})
					}
					if ip.Hostname != "" {
						loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
							Hostname: ip.Hostname,
						})
					}
				}
			}
		}
	}

	// update the load balancer status
	if len(loadBalancerStatus.Ingress) > 0 && !reflect.DeepEqual(ingress.Status.LoadBalancer, loadBalancerStatus) {
		ingress.Status.LoadBalancer = loadBalancerStatus
		return r.Status().Update(ctx, ingress)
	}

	return nil
}

// processIngressClassParameters processes the IngressClass parameters that reference GatewayProxy
func (r *IngressReconciler) processIngressClassParameters(ctx context.Context, tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	ingressClass, err := r.getIngressClass(ingress)
	if err != nil {
		r.Log.Error(err, "failed to get IngressClass", "name", ingress.Spec.IngressClassName)
		return err
	}

	if ingressClass.Spec.Parameters == nil {
		return nil
	}

	parameters := ingressClass.Spec.Parameters
	// check if the parameters reference GatewayProxy
	if parameters.APIGroup != nil && *parameters.APIGroup == v1alpha1.GroupVersion.Group && parameters.Kind == "GatewayProxy" {
		ns := ingress.GetNamespace()
		if parameters.Namespace != nil {
			ns = *parameters.Namespace
		}

		gatewayProxy := &v1alpha1.GatewayProxy{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: ns,
			Name:      parameters.Name,
		}, gatewayProxy); err != nil {
			r.Log.Error(err, "failed to get GatewayProxy", "namespace", ns, "name", parameters.Name)
			return err
		}

		r.Log.Info("found GatewayProxy for IngressClass", "ingressClass", ingressClass.Name, "gatewayproxy", gatewayProxy.Name)
		tctx.GatewayProxies = append(tctx.GatewayProxies, *gatewayProxy)

		// check if the provider field references a secret
		if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane {
			if gatewayProxy.Spec.Provider.ControlPlane != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

				secretRef := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
				secret := &corev1.Secret{}
				if err := r.Get(ctx, client.ObjectKey{
					Namespace: ns,
					Name:      secretRef.Name,
				}, secret); err != nil {
					r.Log.Error(err, "failed to get secret for GatewayProxy provider",
						"namespace", ns,
						"name", secretRef.Name)
					return err
				}

				r.Log.Info("found secret for GatewayProxy provider",
					"ingressClass", ingressClass.Name,
					"gatewayproxy", gatewayProxy.Name,
					"secret", secretRef.Name)

				tctx.Secrets[types.NamespacedName{
					Namespace: ns,
					Name:      secretRef.Name,
				}] = secret
			}
		}
	}

	return nil
}
