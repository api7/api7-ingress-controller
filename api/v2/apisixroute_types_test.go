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

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// celSubjectRule is the CEL expression embedded via +kubebuilder:validation:XValidation
// on ApisixRouteHTTPMatchExprSubject. This test validates its correctness.
const celSubjectRule = "self.scope == 'Path' || self.name != ''"

// evalCELSubjectRule evaluates celSubjectRule against a fake subject object.
func evalCELSubjectRule(t *testing.T, scope, name string) bool {
	t.Helper()
	env, err := cel.NewEnv(
		cel.Variable("self", cel.MapType(cel.StringType, cel.StringType)),
	)
	require.NoError(t, err)

	ast, issues := env.Compile(celSubjectRule)
	require.NoError(t, issues.Err())

	prg, err := env.Program(ast)
	require.NoError(t, err)

	out, _, err := prg.Eval(map[string]any{
		"self": map[string]any{"scope": scope, "name": name},
	})
	require.NoError(t, err)
	return out.Value().(bool)
}

func TestCEL_SubjectRule_ValidScopes(t *testing.T) {
	// All non-Path scopes with a non-empty name must pass.
	for _, scope := range []string{ScopeHeader, ScopeQuery, ScopeCookie, ScopeVariable, ScopeBody} {
		assert.True(t, evalCELSubjectRule(t, scope, "field"), "scope=%s with name should pass", scope)
	}
	// Path scope with empty name must pass (name is ignored for Path).
	assert.True(t, evalCELSubjectRule(t, ScopePath, ""), "Path with empty name should pass")
}

func TestCEL_SubjectRule_InvalidEmptyName(t *testing.T) {
	// Non-Path scopes with empty name must fail.
	for _, scope := range []string{ScopeHeader, ScopeQuery, ScopeCookie, ScopeVariable, ScopeBody} {
		assert.False(t, evalCELSubjectRule(t, scope, ""), "scope=%s with empty name should fail", scope)
	}
}

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
