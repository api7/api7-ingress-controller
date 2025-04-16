package v1alpha1

import gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

type PolicyStatus gatewayv1alpha2.PolicyStatus

// +kubebuilder:validation:XValidation:rule="self.kind == 'Service' && self.group == \"\""
type BackendPolicyTargetReferenceWithSectionName gatewayv1alpha2.LocalPolicyTargetReferenceWithSectionName
