// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
)

// jwtAuthValueSchema mirrors the CEL rule on ApisixConsumerJwtAuthValue.
// It must be kept in sync with the +kubebuilder:validation:XValidation marker
// on that type.
var jwtAuthValueSchema = &apiextensions.JSONSchemaProps{
	Type: "object",
	Properties: map[string]apiextensions.JSONSchemaProps{
		"key":                   {Type: "string"},
		"secret":                {Type: "string"},
		"public_key":            {Type: "string"},
		"private_key":           {Type: "string"},
		"algorithm":             {Type: "string"},
		"exp":                   {Type: "integer", Format: "int64"},
		"base64_secret":         {Type: "boolean"},
		"lifetime_grace_period": {Type: "integer", Format: "int64"},
	},
	Required: []string{"key"},
	XValidations: []apiextensions.ValidationRule{
		{
			Rule:    "!has(self.algorithm) || self.algorithm in ['HS256','HS384','HS512'] || (has(self.public_key) && self.public_key != '') || (has(self.private_key) && self.private_key != '')",
			Message: "asymmetric JWT algorithms (RS*/ES*/PS*/EdDSA) require at least one of public_key or private_key",
		},
	},
}

func validateJwtAuthValue(t *testing.T, obj map[string]interface{}) error {
	t.Helper()
	structural, err := structuralschema.NewStructural(jwtAuthValueSchema)
	require.NoError(t, err, "failed to build structural schema")

	celValidator := cel.NewValidator(structural, false, celconfig.PerCallLimit)
	errs, _ := celValidator.Validate(context.Background(), nil, structural, obj, nil, celconfig.RuntimeCELCostBudget)
	if len(errs) > 0 {
		return errs.ToAggregate()
	}
	return nil
}

// TestJwtAuthCEL_SymmetricHS256WithSecret verifies that HS256 + secret
// without private_key passes CEL validation.
func TestJwtAuthCEL_SymmetricHS256WithSecret(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_SymmetricHS384WithSecret verifies that HS384 + secret passes.
func TestJwtAuthCEL_SymmetricHS384WithSecret(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS384",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_SymmetricHS512WithSecret verifies that HS512 + secret passes.
func TestJwtAuthCEL_SymmetricHS512WithSecret(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS512",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_NoAlgorithmDefaultsToSymmetric verifies that omitting
// algorithm (defaults to HS256 server-side) passes CEL validation.
func TestJwtAuthCEL_NoAlgorithmDefaultsToSymmetric(t *testing.T) {
	obj := map[string]interface{}{
		"key":    "my-key",
		"secret": "my-secret",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithPublicKey verifies that RS256 + public_key passes.
func TestJwtAuthCEL_AsymmetricRS256WithPublicKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":        "my-key",
		"public_key": "-----BEGIN PUBLIC KEY-----\nMFww\n-----END PUBLIC KEY-----",
		"algorithm":  "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithPrivateKey verifies that RS256 + private_key passes
// (backward compatibility: existing configurations only have private_key).
func TestJwtAuthCEL_AsymmetricRS256WithPrivateKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":         "my-key",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----",
		"algorithm":   "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithBothKeys verifies that RS256 + both keys passes.
func TestJwtAuthCEL_AsymmetricRS256WithBothKeys(t *testing.T) {
	obj := map[string]interface{}{
		"key":         "my-key",
		"public_key":  "-----BEGIN PUBLIC KEY-----\nMFww\n-----END PUBLIC KEY-----",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----",
		"algorithm":   "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithoutAnyKey verifies that RS256 without
// any key is rejected by CEL validation.
func TestJwtAuthCEL_AsymmetricRS256WithoutAnyKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "RS256",
	}
	err := validateJwtAuthValue(t, obj)
	assert.Error(t, err, "RS256 without public_key or private_key should be rejected")
	assert.Contains(t, err.Error(), "asymmetric JWT algorithms")
}

// TestJwtAuthCEL_AsymmetricES256WithoutAnyKey verifies that ES256 without
// any key is rejected.
func TestJwtAuthCEL_AsymmetricES256WithoutAnyKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "ES256",
	}
	err := validateJwtAuthValue(t, obj)
	assert.Error(t, err, "ES256 without public_key or private_key should be rejected")
}

// TestJwtAuthCEL_AsymmetricEdDSAWithoutAnyKey verifies that EdDSA without
// any key is rejected.
func TestJwtAuthCEL_AsymmetricEdDSAWithoutAnyKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "EdDSA",
	}
	err := validateJwtAuthValue(t, obj)
	assert.Error(t, err, "EdDSA without public_key or private_key should be rejected")
}

// TestJwtAuthCEL_AsymmetricWithEmptyPublicKey verifies that an asymmetric
// algorithm with an empty public_key string is rejected (same as absent).
func TestJwtAuthCEL_AsymmetricWithEmptyPublicKey(t *testing.T) {
	obj := map[string]interface{}{
		"key":        "my-key",
		"public_key": "",
		"algorithm":  "RS256",
	}
	err := validateJwtAuthValue(t, obj)
	assert.Error(t, err, "RS256 with empty public_key should be rejected")
}
