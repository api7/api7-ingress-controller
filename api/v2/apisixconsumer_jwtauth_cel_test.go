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
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	sigsyaml "sigs.k8s.io/yaml"
)

// loadJwtAuthValueSchema reads the ApisixConsumer CRD YAML and extracts the
// structural schema for spec.authParameter.jwtAuth.value, so that CEL tests
// always validate against the real generated schema rather than a hand-written copy.
func loadJwtAuthValueSchema(t *testing.T) *structuralschema.Structural {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	crdPath := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"config", "crd", "bases", "apisix.apache.org_apisixconsumers.yaml")

	data, err := os.ReadFile(crdPath)
	require.NoError(t, err, "failed to read CRD file: %s", crdPath)

	var crd apiextensionsv1.CustomResourceDefinition
	jsonData, err := sigsyaml.YAMLToJSON(data)
	require.NoError(t, err, "failed to convert CRD YAML to JSON")
	err = json.Unmarshal(jsonData, &crd)
	require.NoError(t, err, "failed to unmarshal CRD")

	// Find the v2 version schema.
	var v1Schema *apiextensionsv1.JSONSchemaProps
	for _, v := range crd.Spec.Versions {
		if v.Name == "v2" {
			v1Schema = v.Schema.OpenAPIV3Schema
			break
		}
	}
	require.NotNil(t, v1Schema, "v2 schema not found in CRD")

	// Navigate: spec.authParameter.jwtAuth.value
	jwtAuthValueV1 := v1Schema.
		Properties["spec"].
		Properties["authParameter"].
		Properties["jwtAuth"].
		Properties["value"]

	// Convert v1 JSONSchemaProps to internal type required by NewStructural.
	var internal apiextensions.JSONSchemaProps
	err = apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
		&jwtAuthValueV1, &internal, nil,
	)
	require.NoError(t, err, "failed to convert v1 schema to internal")

	structural, err := structuralschema.NewStructural(&internal)
	require.NoError(t, err, "failed to build structural schema")
	return structural
}

func validateJwtAuthValue(t *testing.T, structural *structuralschema.Structural, obj map[string]interface{}) error {
	t.Helper()
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
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_SymmetricHS384WithSecret verifies that HS384 + secret passes.
func TestJwtAuthCEL_SymmetricHS384WithSecret(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS384",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_SymmetricHS512WithSecret verifies that HS512 + secret passes.
func TestJwtAuthCEL_SymmetricHS512WithSecret(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"secret":    "my-secret",
		"algorithm": "HS512",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_NoAlgorithmDefaultsToSymmetric verifies that omitting
// algorithm (defaults to HS256 server-side) passes CEL validation.
func TestJwtAuthCEL_NoAlgorithmDefaultsToSymmetric(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":    "my-key",
		"secret": "my-secret",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithPublicKey verifies that RS256 + public_key passes.
func TestJwtAuthCEL_AsymmetricRS256WithPublicKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":        "my-key",
		"public_key": "-----BEGIN PUBLIC KEY-----\nMFww\n-----END PUBLIC KEY-----",
		"algorithm":  "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithPrivateKey verifies that RS256 + private_key passes
// (backward compatibility: existing configurations may only have private_key).
func TestJwtAuthCEL_AsymmetricRS256WithPrivateKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":         "my-key",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----",
		"algorithm":   "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithBothKeys verifies that RS256 + both keys passes.
func TestJwtAuthCEL_AsymmetricRS256WithBothKeys(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":         "my-key",
		"public_key":  "-----BEGIN PUBLIC KEY-----\nMFww\n-----END PUBLIC KEY-----",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----",
		"algorithm":   "RS256",
	}
	assert.NoError(t, validateJwtAuthValue(t, schema, obj))
}

// TestJwtAuthCEL_AsymmetricRS256WithoutAnyKey verifies that RS256 without
// any key is rejected by CEL validation.
func TestJwtAuthCEL_AsymmetricRS256WithoutAnyKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "RS256",
	}
	err := validateJwtAuthValue(t, schema, obj)
	assert.Error(t, err, "RS256 without public_key or private_key should be rejected")
	assert.Contains(t, err.Error(), "asymmetric JWT algorithms")
}

// TestJwtAuthCEL_AsymmetricES256WithoutAnyKey verifies that ES256 without
// any key is rejected.
func TestJwtAuthCEL_AsymmetricES256WithoutAnyKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "ES256",
	}
	err := validateJwtAuthValue(t, schema, obj)
	assert.Error(t, err, "ES256 without public_key or private_key should be rejected")
}

// TestJwtAuthCEL_AsymmetricEdDSAWithoutAnyKey verifies that EdDSA without
// any key is rejected.
func TestJwtAuthCEL_AsymmetricEdDSAWithoutAnyKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":       "my-key",
		"algorithm": "EdDSA",
	}
	err := validateJwtAuthValue(t, schema, obj)
	assert.Error(t, err, "EdDSA without public_key or private_key should be rejected")
}

// TestJwtAuthCEL_AsymmetricWithEmptyPublicKey verifies that an asymmetric
// algorithm with an empty public_key string is rejected (same as absent).
func TestJwtAuthCEL_AsymmetricWithEmptyPublicKey(t *testing.T) {
	schema := loadJwtAuthValueSchema(t)
	obj := map[string]interface{}{
		"key":        "my-key",
		"public_key": "",
		"algorithm":  "RS256",
	}
	err := validateJwtAuthValue(t, schema, obj)
	assert.Error(t, err, "RS256 with empty public_key should be rejected")
}
