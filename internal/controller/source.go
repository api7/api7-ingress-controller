package controller

/*
import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type TargetRefsChangedPredicate struct {
	predicate.Funcs
	DeletedRefs map[string]struct{} // 存储被删除的引用
}

// Update 事件处理
func (t *TargetRefsChangedPredicate) Update(e event.UpdateEvent) bool {
	oldPolicy, okOld := e.ObjectOld.(*v1alpha1.BackendTrafficPolicy)
	newPolicy, okNew := e.ObjectNew.(*v1alpha1.BackendTrafficPolicy)
	if !okOld || !okNew {
		return false
	}

	// 计算被删除的引用
	oldRefs := oldPolicy.Spec.TargetRefs
	newRefs := newPolicy.Spec.TargetRefs

	// 将旧引用转换为 Map
	oldRefMap := make(map[string]struct{})
	for _, ref := range oldRefs {
		key := fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
		oldRefMap[key] = struct{}{}
	}

	// 找出被删除的引用
	t.DeletedRefs = make(map[string]struct{})
	for _, ref := range newRefs {
		key := fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
		delete(oldRefMap, key)
	}
	for key := range oldRefMap {
		t.DeletedRefs[key] = struct{}{}
	}

	return len(t.DeletedRefs) > 0
}

type DeletedRefEventSource struct {
	Client    client.Client
	Predicate *TargetRefsChangedPredicate
}

// Start 实现 Source 接口
func (s *DeletedRefEventSource) Start(
	ctx context.Context,
	handler handler.EventHandler,
	queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate,
) error {
	// 监听 BackendTrafficPolicy 的 Update 事件（已通过 Predicate 过滤）
	// 此处假设 Predicate 已捕获到被删除的引用
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if len(s.Predicate.DeletedRefs) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				// 遍历被删除的引用，查找关联的 HTTPRoute
				for refKey := range s.Predicate.DeletedRefs {
					parts := strings.Split(refKey, "/")
					if len(parts) != 2 {
						continue
					}
					kind, name := parts[0], parts[1]

					// 查找关联的 HTTPRoute
					var routes gatewayv1.HTTPRouteList
					if err := s.Client.List(
						context.Background(),
						&routes,
						client.MatchingFields{"targetRefs": refKey},
					); err != nil {
						continue
					}

					// 生成调和请求
					for _, route := range routes.Items {
						req := reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      route.Name,
								Namespace: route.Namespace,
							},
						}
						handler.Generic(ctx, event.GenericEvent{Object: &route}, queue)
					}
				}

				// 清空已处理的引用
				s.Predicate.DeletedRefs = make(map[string]struct{})
				time.Sleep(5 * time.Second) // 避免高频触发
			}
		}
	}()
	return nil
}
*/
