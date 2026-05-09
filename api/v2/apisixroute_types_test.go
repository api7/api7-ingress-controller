// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestToVars_ScopeBody_SimpleField(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "action",
			},
			Op:    OpEqual,
			Value: strPtr("login"),
		},
	}

	vars, err := exprs.ToVars()
	require.NoError(t, err)
	require.Len(t, vars, 1)

	// vars[0] is []StringOrSlice: [subject, op, value]
	// Should map to post_arg.action
	assert.Equal(t, "post_arg.action", vars[0][0].StrVal)
	assert.Equal(t, "==", vars[0][1].StrVal)
	assert.Equal(t, "login", vars[0][2].StrVal)
}

func TestToVars_ScopeBody_NestedJSONPath(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "model.version",
			},
			Op:    OpEqual,
			Value: strPtr("gpt-4"),
		},
	}

	vars, err := exprs.ToVars()
	require.NoError(t, err)
	require.Len(t, vars, 1)

	// Should map to post_arg.model.version (dot-notation passthrough)
	assert.Equal(t, "post_arg.model.version", vars[0][0].StrVal)
}

func TestToVars_ScopeBody_EmptyName_ReturnsError(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "",
			},
			Op:    OpEqual,
			Value: strPtr("login"),
		},
	}

	_, err := exprs.ToVars()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty subject.name")
}
