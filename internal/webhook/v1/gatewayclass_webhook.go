package v1

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _gcLog = logf.Log.WithName("gatewayclass-resource")

// SetupGatewayClassWebhookWithManager registers the webhook for GatewayClass in the manager.
func SetupGatewayClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gatewayv1.GatewayClass{}).
		WithValidator(&GatewayClassCustomValidator{
			Client: mgr.GetClient(),
			Logger: _gcLog,
		}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1-gatewayclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=create;update;delete,versions=v1,name=vgatewayclass-v1.kb.io,admissionReviewVersions=v1

// GatewayClassCustomValidator struct is responsible for validating the GatewayClass resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GatewayClassCustomValidator struct {
	client.Client
	logr.Logger
}

var _ webhook.CustomValidator = &GatewayClassCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GatewayClass.
func (v *GatewayClassCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GatewayClass.
func (v *GatewayClassCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GatewayClass.
func (v *GatewayClassCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gc, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		return nil, fmt.Errorf("expected a GatewayClass object but got %T", obj)
	}
	v.Info("Validation for GatewayClass upon deletion", "name", gc.GetName())

	var gatewayList gatewayv1.GatewayList
	if err := v.List(ctx, &gatewayList); err != nil {
		v.Error(err, "failed to list gateway for the GatewayClass")
		return nil, err
	}
	var gateways []types.NamespacedName
	for _, gateway := range gatewayList.Items {
		if string(gateway.Spec.GatewayClassName) == gc.Name {
			gateways = append(gateways, types.NamespacedName{
				Namespace: gateway.GetNamespace(),
				Name:      gateway.GetName(),
			})
		}
	}
	if len(gateways) > 0 {
		err := fmt.Errorf("the GatewayClass is still in using by Gateways: %v", gateways)
		v.Error(err, "can not delete the GatewayClass")
		return nil, err
	}

	return nil, nil
}
