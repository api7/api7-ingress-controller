package controller

import (
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

const (
	ConditionTypeAvailable   string = "Available"
	ConditionTypeProgressing string = "Progressing"
	ConditionTypeDegraded    string = "Degraded"

	ConditionReasonSynced    string = "ResourceSynced"
	ConditionReasonSyncAbort string = "ResourceSyncAbort"
)

func NewCondition(observedGeneration int64, status bool, message string) metav1.Condition {
	condition := metav1.ConditionTrue
	reason := ConditionReasonSynced
	if !status {
		condition = metav1.ConditionFalse
		reason = ConditionReasonSyncAbort
	}
	return metav1.Condition{
		Type:               ConditionTypeAvailable,
		Reason:             reason,
		Status:             condition,
		Message:            message,
		ObservedGeneration: observedGeneration,
	}
}

func VerifyConditions(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		return true
	}

	if existingCondition.ObservedGeneration > newCondition.ObservedGeneration {
		return false
	}
	if *existingCondition == newCondition {
		return false
	}
	return true
}

func NewPolicyCondition(observedGeneration int64, status bool, message string) metav1.Condition {
	conditionStatus := metav1.ConditionTrue
	reason := string(gatewayv1alpha2.PolicyReasonAccepted)
	if !status {
		conditionStatus = metav1.ConditionFalse
		reason = string(gatewayv1alpha2.PolicyReasonInvalid)
	}

	return metav1.Condition{
		Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
		Reason:             reason,
		Status:             conditionStatus,
		Message:            message,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
	}
}

func NewPolicyConflictCondition(observedGeneration int64, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
		Reason:             string(gatewayv1alpha2.PolicyReasonConflicted),
		Status:             metav1.ConditionFalse,
		Message:            message,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
	}
}

func UpdateStatus(
	c client.Client,
	log logr.Logger,
	tctx *provider.TranslateContext,
) {
	for _, obj := range tctx.StatusUpdaters {
		_ = c.Status().Update(tctx, obj)
	}
}
