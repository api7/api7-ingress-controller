package lister

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	types "github.com/apache/apisix-ingress-controller/internal/types"
)

// GenericIngressClass 是对 v1 和 v1beta1 的通用抽象
type IngressClass struct {
	Name       string
	Controller string
	Raw        client.Object
}

// IngressClassLister 定义统一的接口
type IngressClassLister interface {
	List(ctx context.Context) ([]*types.IngressClass, error)
	Get(ctx context.Context, name string) (*types.IngressClass, error)
}

type ingressClassV1Lister struct {
	client client.Client
}

func (l *ingressClassV1Lister) List(ctx context.Context) ([]*types.IngressClass, error) {
	var list networkingv1.IngressClassList
	if err := l.client.List(ctx, &list); err != nil {
		return nil, err
	}

	var result []*types.IngressClass
	for _, item := range list.Items {
		obj := item // copy to avoid pointer issue
		result = append(result, &types.IngressClass{
			Object: &obj,
		})
	}
	return result, nil
}

func (l *ingressClassV1Lister) Get(ctx context.Context, name string) (*types.IngressClass, error) {
	var obj networkingv1.IngressClass
	if err := l.client.Get(ctx, client.ObjectKey{Name: name}, &obj); err != nil {
		return nil, err
	}
	return &types.IngressClass{
		Object: &obj,
	}, nil
}

type ingressClassV1beta1Lister struct {
	client client.Client
}

func (l *ingressClassV1beta1Lister) List(ctx context.Context) ([]*types.IngressClass, error) {
	var list networkingv1beta1.IngressClassList
	if err := l.client.List(ctx, &list); err != nil {
		return nil, err
	}

	var result []*types.IngressClass
	for _, item := range list.Items {
		obj := item
		result = append(result, &types.IngressClass{
			Object: &obj,
		})
	}
	return result, nil
}

func (l *ingressClassV1beta1Lister) Get(ctx context.Context, name string) (*types.IngressClass, error) {
	var obj networkingv1beta1.IngressClass
	if err := l.client.Get(ctx, client.ObjectKey{Name: name}, &obj); err != nil {
		return nil, err
	}
	return &types.IngressClass{
		Object: &obj,
	}, nil
}

func NewIngressClassLister(client client.Client, gvk schema.GroupVersionKind) IngressClassLister {
	switch gvk {
	case networkingv1.SchemeGroupVersion.WithKind("IngressClass"):
		return &ingressClassV1Lister{client: client}
	case networkingv1beta1.SchemeGroupVersion.WithKind("IngressClass"):
		return &ingressClassV1beta1Lister{client: client}
	}
	return nil
}
