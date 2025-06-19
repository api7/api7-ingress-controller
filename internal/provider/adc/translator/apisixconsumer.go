// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translator

import (
	"fmt"
	"strconv"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	v2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/pkg/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

var (
	_errKeyNotFoundOrInvalid      = errors.New("key \"key\" not found or invalid in secret")
	_errUsernameNotFoundOrInvalid = errors.New("key \"username\" not found or invalid in secret")
	_errPasswordNotFoundOrInvalid = errors.New("key \"password\" not found or invalid in secret")

	_jwtAuthExpDefaultValue = int64(868400)

	_hmacAuthAlgorithmDefaultValue           = "hmac-sha256"
	_hmacAuthClockSkewDefaultValue           = int64(0)
	_hmacAuthKeepHeadersDefaultValue         = false
	_hmacAuthEncodeURIParamsDefaultValue     = true
	_hmacAuthValidateRequestBodyDefaultValue = false
	_hmacAuthMaxReqBodyDefaultValue          = int64(524288)

	_stringTrue = "true"
)

func (t *Translator) TranslateApisixConsumer(tctx *provider.TranslateContext, ac *v2.ApisixConsumer) (*TranslateResult, error) {
	result := &TranslateResult{}
	plugins := make(adctypes.Plugins)
	if ac.Spec.AuthParameter.KeyAuth != nil {
		cfg, err := t.translateConsumerKeyAuthPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.KeyAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid key auth config: %s", err)
		}
		plugins["key-auth"] = cfg
	} else if ac.Spec.AuthParameter.BasicAuth != nil {
		cfg, err := t.translateConsumerBasicAuthPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.BasicAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid basic auth config: %s", err)
		}
		plugins["basic-auth"] = cfg
	} else if ac.Spec.AuthParameter.JwtAuth != nil {
		cfg, err := t.translateConsumerJwtAuthPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.JwtAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid jwt auth config: %s", err)
		}
		plugins["jwt-auth"] = cfg
	} else if ac.Spec.AuthParameter.WolfRBAC != nil {
		cfg, err := t.translateConsumerWolfRBACPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.WolfRBAC)
		if err != nil {
			return nil, fmt.Errorf("invalid wolf rbac config: %s", err)
		}
		plugins["wolf-rbac"] = cfg
	} else if ac.Spec.AuthParameter.HMACAuth != nil {
		cfg, err := t.translateConsumerHMACAuthPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.HMACAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid hmac auth config: %s", err)
		}
		plugins["hmac-auth"] = cfg
	} else if ac.Spec.AuthParameter.LDAPAuth != nil {
		cfg, err := t.translateConsumerLDAPAuthPluginV2(tctx, ac.Namespace, ac.Spec.AuthParameter.LDAPAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid ldap auth config: %s", err)
		}
		plugins["ldap-auth"] = cfg
	}

	username := adctypes.ComposeConsumerName(ac.Namespace, ac.Name)
	consumer := &adctypes.Consumer{
		Username: username,
	}
	consumer.Plugins = plugins
	consumer.Labels = label.GenLabel(ac)
	result.Consumers = append(result.Consumers, consumer)
	return result, nil
}

func (t *Translator) translateConsumerKeyAuthPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerKeyAuth) (*types.KeyAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &types.KeyAuthConsumerConfig{Key: cfg.Value.Key}, nil
	}

	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}
	raw, ok := sec.Data["key"]
	if !ok || len(raw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}
	return &types.KeyAuthConsumerConfig{Key: string(raw)}, nil
}

func (t *Translator) translateConsumerBasicAuthPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerBasicAuth) (*types.BasicAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &types.BasicAuthConsumerConfig{
			Username: cfg.Value.Username,
			Password: cfg.Value.Password,
		}, nil
	}

	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}
	raw1, ok := sec.Data["username"]
	if !ok || len(raw1) == 0 {
		return nil, _errUsernameNotFoundOrInvalid
	}
	raw2, ok := sec.Data["password"]
	if !ok || len(raw2) == 0 {
		return nil, _errPasswordNotFoundOrInvalid
	}
	return &types.BasicAuthConsumerConfig{
		Username: string(raw1),
		Password: string(raw2),
	}, nil
}

func (t *Translator) translateConsumerWolfRBACPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerWolfRBAC) (*types.WolfRBACConsumerConfig, error) {
	if cfg.Value != nil {
		return &types.WolfRBACConsumerConfig{
			Server:       cfg.Value.Server,
			Appid:        cfg.Value.Appid,
			HeaderPrefix: cfg.Value.HeaderPrefix,
		}, nil
	}
	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}
	raw1 := sec.Data["server"]
	raw2 := sec.Data["appid"]
	raw3 := sec.Data["header_prefix"]
	return &types.WolfRBACConsumerConfig{
		Server:       string(raw1),
		Appid:        string(raw2),
		HeaderPrefix: string(raw3),
	}, nil
}

func (t *Translator) translateConsumerJwtAuthPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerJwtAuth) (*types.JwtAuthConsumerConfig, error) {
	if cfg.Value != nil {
		// The field exp must be a positive integer, default value 86400.
		if cfg.Value.Exp < 1 {
			cfg.Value.Exp = _jwtAuthExpDefaultValue
		}
		return &types.JwtAuthConsumerConfig{
			Key:                 cfg.Value.Key,
			Secret:              cfg.Value.Secret,
			PublicKey:           cfg.Value.PublicKey,
			PrivateKey:          cfg.Value.PrivateKey,
			Algorithm:           cfg.Value.Algorithm,
			Exp:                 cfg.Value.Exp,
			Base64Secret:        cfg.Value.Base64Secret,
			LifetimeGracePeriod: cfg.Value.LifetimeGracePeriod,
		}, nil
	}

	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}
	keyRaw, ok := sec.Data["key"]
	if !ok || len(keyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}
	base64SecretRaw := sec.Data["base64_secret"]
	var base64Secret bool
	if string(base64SecretRaw) == _stringTrue {
		base64Secret = true
	}
	expRaw := sec.Data["exp"]
	exp, _ := strconv.ParseInt(string(expRaw), 10, 64)
	// The field exp must be a positive integer, default value 86400.
	if exp < 1 {
		exp = _jwtAuthExpDefaultValue
	}
	lifetimeGracePeriodRaw := sec.Data["lifetime_grace_period"]
	lifetimeGracePeriod, _ := strconv.ParseInt(string(lifetimeGracePeriodRaw), 10, 64)
	secretRaw := sec.Data["secret"]
	publicKeyRaw := sec.Data["public_key"]
	privateKeyRaw := sec.Data["private_key"]
	algorithmRaw := sec.Data["algorithm"]

	return &types.JwtAuthConsumerConfig{
		Key:                 string(keyRaw),
		Secret:              string(secretRaw),
		PublicKey:           string(publicKeyRaw),
		PrivateKey:          string(privateKeyRaw),
		Algorithm:           string(algorithmRaw),
		Exp:                 exp,
		Base64Secret:        base64Secret,
		LifetimeGracePeriod: lifetimeGracePeriod,
	}, nil
}

func (t *Translator) translateConsumerHMACAuthPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerHMACAuth) (*types.HMACAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &types.HMACAuthConsumerConfig{
			AccessKey:           cfg.Value.AccessKey,
			SecretKey:           cfg.Value.SecretKey,
			Algorithm:           cfg.Value.Algorithm,
			ClockSkew:           cfg.Value.ClockSkew,
			SignedHeaders:       cfg.Value.SignedHeaders,
			KeepHeaders:         cfg.Value.KeepHeaders,
			EncodeURIParams:     cfg.Value.EncodeURIParams,
			ValidateRequestBody: cfg.Value.ValidateRequestBody,
			MaxReqBody:          cfg.Value.MaxReqBody,
		}, nil
	}

	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}

	accessKeyRaw, ok := sec.Data["access_key"]
	if !ok || len(accessKeyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	secretKeyRaw, ok := sec.Data["secret_key"]
	if !ok || len(secretKeyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	algorithmRaw, ok := sec.Data["algorithm"]
	var algorithm string
	if !ok {
		algorithm = _hmacAuthAlgorithmDefaultValue
	} else {
		algorithm = string(algorithmRaw)
	}

	clockSkewRaw := sec.Data["clock_skew"]
	clockSkew, _ := strconv.ParseInt(string(clockSkewRaw), 10, 64)
	if clockSkew < 0 {
		clockSkew = _hmacAuthClockSkewDefaultValue
	}

	signedHeadersRaw := sec.Data["signed_headers"]
	signedHeaders := make([]string, 0, len(signedHeadersRaw))
	for _, b := range signedHeadersRaw {
		signedHeaders = append(signedHeaders, string(b))
	}

	var keepHeader bool
	keepHeaderRaw, ok := sec.Data["keep_headers"]
	if !ok {
		keepHeader = _hmacAuthKeepHeadersDefaultValue
	} else {
		if string(keepHeaderRaw) == _stringTrue {
			keepHeader = true
		} else {
			keepHeader = false
		}
	}

	var encodeURIParams bool
	encodeURIParamsRaw, ok := sec.Data["encode_uri_params"]
	if !ok {
		encodeURIParams = _hmacAuthEncodeURIParamsDefaultValue
	} else {
		if string(encodeURIParamsRaw) == _stringTrue {
			encodeURIParams = true
		} else {
			encodeURIParams = false
		}
	}

	var validateRequestBody bool
	validateRequestBodyRaw, ok := sec.Data["validate_request_body"]
	if !ok {
		validateRequestBody = _hmacAuthValidateRequestBodyDefaultValue
	} else {
		if string(validateRequestBodyRaw) == _stringTrue {
			validateRequestBody = true
		} else {
			validateRequestBody = false
		}
	}

	maxReqBodyRaw := sec.Data["max_req_body"]
	maxReqBody, _ := strconv.ParseInt(string(maxReqBodyRaw), 10, 64)
	if maxReqBody < 0 {
		maxReqBody = _hmacAuthMaxReqBodyDefaultValue
	}

	return &types.HMACAuthConsumerConfig{
		AccessKey:           string(accessKeyRaw),
		SecretKey:           string(secretKeyRaw),
		Algorithm:           algorithm,
		ClockSkew:           clockSkew,
		SignedHeaders:       signedHeaders,
		KeepHeaders:         keepHeader,
		EncodeURIParams:     encodeURIParams,
		ValidateRequestBody: validateRequestBody,
		MaxReqBody:          maxReqBody,
	}, nil
}

func (t *Translator) translateConsumerLDAPAuthPluginV2(tctx *provider.TranslateContext, consumerNamespace string, cfg *v2.ApisixConsumerLDAPAuth) (*types.LDAPAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &types.LDAPAuthConsumerConfig{
			UserDN: cfg.Value.UserDN,
		}, nil
	}

	sec := tctx.Secrets[k8stypes.NamespacedName{
		Namespace: consumerNamespace,
		Name:      cfg.SecretRef.Name,
	}]
	if sec == nil {
		return nil, fmt.Errorf("secret %s/%s not found", consumerNamespace, cfg.SecretRef.Name)
	}
	userDNRaw, ok := sec.Data["user_dn"]
	if !ok || len(userDNRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	return &types.LDAPAuthConsumerConfig{
		UserDN: string(userDNRaw),
	}, nil
}
