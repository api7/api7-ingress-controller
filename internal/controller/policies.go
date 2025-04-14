package controller

import (
	"fmt"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PolicyTargetKey struct {
	NsName    types.NamespacedName
	GroupKind schema.GroupKind
}

func (p PolicyTargetKey) String() string {
	return p.NsName.String() + "/" + p.GroupKind.String()
}

func processPolicy(c client.Client, tctx *provider.TranslateContext) {
}

func processBackendTrafficPolicy(c client.Client, tctx *provider.TranslateContext) {
	conflicts := map[string]v1alpha1.BackendTrafficPolicy{}
	for _, service := range tctx.Services {
		backendTrafficPolicyList := &v1alpha1.BackendTrafficPolicyList{}
		if err := c.List(tctx, backendTrafficPolicyList,
			client.MatchingFields{
				indexer.PolicyTargetRefs: indexer.GenIndexKeyWithGK("", "Service", service.Namespace, service.Name),
			},
		); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return
		}
		if len(backendTrafficPolicyList.Items) == 0 {
			continue
		}

		portNameExist := make(map[string]bool, len(service.Spec.Ports))
		for _, port := range service.Spec.Ports {
			portNameExist[port.Name] = true
		}
		for _, policy := range backendTrafficPolicyList.Items {
			targetRefs := policy.Spec.TargetRefs
			for _, targetRef := range targetRefs {
				sectionName := targetRef.SectionName
				key := PolicyTargetKey{
					NsName:    types.NamespacedName{Namespace: service.Namespace, Name: service.Name},
					GroupKind: schema.GroupKind{Group: "", Kind: "Service"},
				}
				condition := NewPolicyCondition(policy.Generation, true, "Policy has been accepted")
				if sectionName != nil && !portNameExist[string(*sectionName)] {
					condition = NewPolicyCondition(policy.Generation, false, fmt.Sprintf("SectionName %s not found in Service %s/%s", *sectionName, service.Namespace, service.Name))
					goto record_status
				}
				if p, ok := conflicts[key.String()]; ok && (p.Name == policy.Name && p.Namespace != policy.Namespace) {
					condition = NewPolicyConflictCondition(p.Generation, fmt.Sprintf("Unable to target Service %s/%s with BackendTrafficPolicy %s/%s", service.Namespace, service.Name, policy.Namespace, policy.Name))
					goto record_status
				}
				conflicts[key.String()] = policy
			record_status:
				if ok := SetAncestors(&policy.Status, tctx.ParentRefs, condition); ok {
					tctx.StatusUpdaters = append(tctx.StatusUpdaters, &policy)
				}
			}
			policy.GetGenerateName()

			tctx.BackendTrafficPolicies[types.NamespacedName{
				Name:      policy.Name,
				Namespace: policy.Namespace,
			}] = &policy
		}
	}
}
func SetAncestors(status *v1alpha1.PolicyStatus, parentRefs []gatewayv1.ParentReference, condition metav1.Condition) bool {
	updated := false
	for _, parent := range parentRefs {
		ancestorStatus := gatewayv1alpha2.PolicyAncestorStatus{
			AncestorRef:    parent,
			Conditions:     []metav1.Condition{condition},
			ControllerName: gatewayv1alpha2.GatewayController(config.ControllerConfig.ControllerName),
		}
		if SetAncestorStatus(status, ancestorStatus) {
			updated = true
		}
	}
	return updated
}

func SetAncestorStatus(status *v1alpha1.PolicyStatus, ancestorStatus gatewayv1alpha2.PolicyAncestorStatus) bool {
	for _, c := range status.Ancestors {
		if c.AncestorRef == ancestorStatus.AncestorRef {
			if c.Conditions[0].ObservedGeneration < ancestorStatus.Conditions[0].ObservedGeneration {
				c.Conditions = ancestorStatus.Conditions
				return true
			}
			return false
		}
	}
	status.Ancestors = append(status.Ancestors, ancestorStatus)
	return true
}
