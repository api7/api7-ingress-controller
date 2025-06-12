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
	"net"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func (t *Translator) TranslateApisixRoute(tctx *provider.TranslateContext, obj *apiv2.ApisixRoute) (*TranslateResult, error) {

	return nil, errors.New("not implemented yet")
}

func TranslateApisixRouteVars(exprs []apiv2.ApisixRouteHTTPMatchExpr) (result adc.StringOrSlice, err error) {
	for _, expr := range exprs {
		if expr.Subject.Name == "" && expr.Subject.Scope != apiv2.ScopePath {
			return result, errors.New("empty subject.name")
		}

		// process key
		var (
			subj string
			this adc.StringOrSlice
		)
		switch expr.Subject.Scope {
		case apiv2.ScopeQuery:
			subj = "arg_" + expr.Subject.Name
		case apiv2.ScopeHeader:
			subj = "http_" + strings.ReplaceAll(strings.ToLower(expr.Subject.Name), "-", "_")
		case apiv2.ScopeCookie:
			subj = "cookie_" + expr.Subject.Name
		case apiv2.ScopePath:
			subj = "uri"
		case apiv2.ScopeVariable:
			subj = expr.Subject.Name
		default:
			return result, errors.New("invalid http match expr: subject.scope should be one of [query, header, cookie, path, variable]")
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: subj})

		// process operator
		var (
			op string
		)
		switch expr.Op {
		case apiv2.OpEqual:
			op = "=="
		case apiv2.OpGreaterThan:
			op = ">"
		case apiv2.OpGreaterThanEqual:
			op = ">="
		case apiv2.OpIn:
			op = "in"
		case apiv2.OpLessThan:
			op = "<"
		case apiv2.OpLessThanEqual:
			op = "<="
		case apiv2.OpNotEqual:
			op = "~="
		case apiv2.OpNotIn:
			op = "in"
		case apiv2.OpRegexMatch:
			op = "~~"
		case apiv2.OpRegexMatchCaseInsensitive:
			op = "~*"
		case apiv2.OpRegexNotMatch:
			op = "~~"
		case apiv2.OpRegexNotMatchCaseInsensitive:
			op = "~*"
		default:
			return result, errors.New("unknown operator")
		}
		if invert := slices.Contains([]string{apiv2.OpNotIn, apiv2.OpRegexNotMatch, apiv2.OpRegexNotMatchCaseInsensitive}, op); invert {
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: "!"})
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: op})

		// process value
		switch expr.Op {
		case apiv2.OpIn, apiv2.OpNotIn:
			if expr.Set == nil {
				return result, errors.New("empty set value")
			}
			var value adc.StringOrSlice
			for _, item := range expr.Set {
				value.SliceVal = append(value.SliceVal, adc.StringOrSlice{StrVal: item})
			}
			this.SliceVal = append(this.SliceVal, value)
		default:
			if expr.Value == nil {
				return result, errors.New("empty value")
			}
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: *expr.Value})
		}

		// append to result
		result.SliceVal = append(result.SliceVal, this)
	}

	return result, nil
}

func ValidateRemoteAddrs(remoteAddrs []string) error {
	for _, addr := range remoteAddrs {
		if ip := net.ParseIP(addr); ip == nil {
			// addr is not an IP address, try to parse it as a CIDR.
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return err
			}
		}
	}
	return nil
}
