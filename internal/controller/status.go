package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTypeAvailable   string = "Available"
	ConditionTypeProgressing string = "Progressing"
	ConditionTypeDegraded    string = "Degraded"

	ConditionReasonSynced string = "ResourceSynced"
)

func NewCondition(observedGeneration int64, status bool, message string) metav1.Condition {
	condition := metav1.ConditionTrue
	if !status {
		condition = metav1.ConditionFalse
	}
	return metav1.Condition{
		Type:               ConditionTypeAvailable,
		Reason:             ConditionReasonSynced,
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
