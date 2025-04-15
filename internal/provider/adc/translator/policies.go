package translator

import (
	"github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
)

func (t *Translator) AttachBackendTrafficPolicyToUpstream(policies map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if len(policies) == 0 {
		return
	}
	for _, policy := range policies {
		t.attachBackendTrafficPolicyToUpstream(policy, upstream)
	}

}

func (t *Translator) attachBackendTrafficPolicyToUpstream(policy *v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if policy == nil {
		return
	}
	upstream.UpstreamHost = string(policy.Spec.Host)
	upstream.Scheme = policy.Spec.Scheme
	if policy.Spec.Retries != nil {
		upstream.Retries = new(int64)
		*upstream.Retries = int64(*policy.Spec.Retries)
	}
	if policy.Spec.Timeout != nil {
		var (
			connect *int64
			read    *int64
			send    *int64
		)
		if policy.Spec.Timeout.Connect.Duration > 0 {
			connect = new(int64)
			*connect = policy.Spec.Timeout.Connect.Duration.Milliseconds()
		}
		if policy.Spec.Timeout.Read.Duration > 0 {
			read = new(int64)
			*read = policy.Spec.Timeout.Read.Duration.Milliseconds()
		}
		if policy.Spec.Timeout.Send.Duration > 0 {
			send = new(int64)
			*send = policy.Spec.Timeout.Send.Duration.Milliseconds()
		}
		upstream.Timeout = &adctypes.Timeout{
			Connect: connect,
			Read:    read,
			Send:    send,
		}
	}
	if policy.Spec.LoadBalancer != nil {
		upstream.Type = adc.UpstreamType(policy.Spec.LoadBalancer.Type)
		upstream.HashOn = policy.Spec.LoadBalancer.HashOn
		upstream.Key = policy.Spec.LoadBalancer.Key
	}
}
