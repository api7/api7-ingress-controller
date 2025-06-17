// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"cmp"
	"errors"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func (t *Translator) TranslateApisixUpstream(au *apiv2.ApisixUpstream) (ups *adc.Upstream, err error) {
	ups = adc.NewDefaultUpstream()
	for _, f := range []func(*apiv2.ApisixUpstream, *adc.Upstream) error{
		translateApisixUpstreamScheme,
		translateApisixUpstreamLoadBalancer,
		translateApisixUpstreamHealthCheck,
		translateApisixUpstreamRetriesAndTimeout,
		translateApisixUpstreamClientTLS,
		translateApisixUpstreamPassHost,
		translateApisixUpstreamDiscovery,
	} {
		if err = f(au, ups); err != nil {
			return
		}
	}

	return
}

func translateApisixUpstreamScheme(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	switch au.Spec.Scheme {
	case apiv2.SchemeHTTP, apiv2.SchemeHTTPS, apiv2.SchemeGRPC, apiv2.SchemeGRPCS:
		ups.Scheme = au.Spec.Scheme
	default:
		ups.Scheme = apiv2.SchemeHTTP
	}
	return nil
}

func translateApisixUpstreamLoadBalancer(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	lb := au.Spec.LoadBalancer
	if lb == nil || lb.Type == "" {
		ups.Type = apiv2.LbRoundRobin
		return nil
	}
	switch lb.Type {
	case apiv2.LbRoundRobin, apiv2.LbLeastConn, apiv2.LbEwma:
		ups.Type = adc.UpstreamType(lb.Type)
	case apiv2.LbConsistentHash:
		ups.Type = adc.UpstreamType(lb.Type)
		ups.Key = lb.Key
		switch lb.HashOn {
		case apiv2.HashOnVars:
			fallthrough
		case apiv2.HashOnHeader:
			fallthrough
		case apiv2.HashOnCookie:
			fallthrough
		case apiv2.HashOnConsumer:
			fallthrough
		case apiv2.HashOnVarsCombination:
			ups.HashOn = lb.HashOn
		default:
			return errors.New("invalid hashOn")
		}
	default:
		return errors.New("invalid loadBalancer type")
	}
	return nil
}

func translateApisixUpstreamHealthCheck(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	// todo: no field `.Checks` in adc.Upstream
	return nil
}

func translateApisixUpstreamRetriesAndTimeout(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	retries := au.Spec.Retries
	timeout := au.Spec.Timeout

	if retries != nil && *retries < 0 {
		return errors.New("invalid value retries")
	}
	ups.Retries = retries

	if timeout == nil {
		return nil
	}
	if timeout.Connect.Duration < 0 {
		return errors.New("invalid value timeout.connect")
	}
	if timeout.Read.Duration < 0 {
		return errors.New("invalid value timeout.read")
	}
	if timeout.Send.Duration < 0 {
		return errors.New("invalid value timeout.send")
	}

	// Since the schema of timeout doesn't allow only configuring
	// one or two items. Here we assign the default value first.
	connTimeout := cmp.Or(timeout.Connect.Duration, apiv2.DefaultUpstreamTimeout)
	readTimeout := cmp.Or(timeout.Read.Duration, apiv2.DefaultUpstreamTimeout)
	sendTimeout := cmp.Or(timeout.Send.Duration, apiv2.DefaultUpstreamTimeout)

	ups.Timeout = &adc.Timeout{
		Connect: int(connTimeout.Seconds()),
		Read:    int(readTimeout.Seconds()),
		Send:    int(sendTimeout.Seconds()),
	}

	return nil
}

func translateApisixUpstreamClientTLS(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	// todo: no field `.TLS` in adc.Upstream
	return nil
}

func translateApisixUpstreamPassHost(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	switch passHost := au.Spec.PassHost; passHost {
	case apiv2.PassHostPass, apiv2.PassHostNode, apiv2.PassHostRewrite:
		ups.PassHost = passHost
	default:
		ups.PassHost = ""
	}

	ups.UpstreamHost = au.Spec.UpstreamHost

	return nil
}

func translateApisixUpstreamDiscovery(upstream *apiv2.ApisixUpstream, upstream2 *adc.Upstream) error {
	// todo: no filed `.Discovery*` in adc.Upstream
	return nil
}
