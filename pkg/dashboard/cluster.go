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

package dashboard

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/pkg/dashboard/cache"
	"github.com/api7/gopkg/pkg/log"
)

const (
	_defaultTimeout      = 5 * time.Second
	_defaultSyncInterval = 6 * time.Hour

	_cacheSyncing = iota
	_cacheSynced
)

var (
	// ErrClusterNotExist means a cluster doesn't exist.
	ErrClusterNotExist = errors.New("cluster not exist")
	// ErrDuplicatedCluster means the cluster adding request was
	// rejected since the cluster was already created.
	ErrDuplicatedCluster = errors.New("duplicated cluster")
	// ErrFunctionDisabled means the APISIX function is disabled
	ErrFunctionDisabled = errors.New("function disabled")

	DefaultLabelsManagedBy map[string]string = map[string]string{
		"managed-by": "api7-ingress-controller",
	}

	// ErrRouteNotFound means the [route, ssl, upstream] was not found.
	ErrNotFound = cache.ErrNotFound

	_errReadOnClosedResBody = errors.New("http: read on closed response body")

	// Default shared transport for apisix client
	_defaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).Dial,
		DialContext: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
)

// ClusterOptions contains parameters to customize APISIX client.
type ClusterOptions struct {
	AdminAPIVersion string
	Name            string
	AdminKey        string
	BaseURL         string
	Timeout         time.Duration
	// SyncInterval is the interval to sync schema.
	SyncComparison    bool
	EnableEtcdServer  bool
	Prefix            string
	ListenAddress     string
	SchemaSynced      bool
	CacheSynced       bool
	SSLKeyEncryptSalt string
}

type cluster struct {
	adminVersion      string
	name              string
	baseURL           string
	baseURLHost       string
	adminKey          string
	prefix            string
	cli               *http.Client
	cacheState        int32
	cache             cache.Cache
	cacheSynced       chan struct{}
	cacheSyncErr      error
	route             Route
	service           Service
	ssl               SSL
	streamRoute       StreamRoute
	globalRules       GlobalRule
	consumer          Consumer
	plugin            Plugin
	schema            Schema
	pluginConfig      PluginConfig
	pluginMetadata    PluginMetadata
	waitforCacheSync  bool
	validator         APISIXSchemaValidator
	sslKeyEncryptSalt string
}

func newCluster(ctx context.Context, o *ClusterOptions) (Cluster, error) {
	if o.BaseURL == "" {
		return nil, errors.New("empty base url")
	}
	if o.Timeout == time.Duration(0) {
		o.Timeout = _defaultTimeout
	}
	o.BaseURL = strings.TrimSuffix(o.BaseURL, "/")

	u, err := url.Parse(o.BaseURL)
	if err != nil {
		return nil, err
	}
	if u.Port() == "" {
		if u.Scheme == "http" {
			u.Host = u.Host + ":80"
		} else if u.Scheme == "https" {
			u.Host = u.Host + ":443"
		}
	}

	// if the version is not v3, then fallback to v2
	adminVersion := o.AdminAPIVersion
	c := &cluster{
		adminVersion: adminVersion,
		name:         o.Name,
		baseURL:      o.BaseURL,
		baseURLHost:  u.Host,
		adminKey:     o.AdminKey,
		prefix:       o.Prefix,
		cli: &http.Client{
			Timeout:   o.Timeout,
			Transport: _defaultTransport,
		},
		cacheState:        _cacheSyncing, // default state
		cacheSynced:       make(chan struct{}),
		sslKeyEncryptSalt: o.SSLKeyEncryptSalt,
	}

	c.route = newRouteClient(c)
	c.service = newServiceClient(c)
	c.ssl = newSSLClient(c)
	c.streamRoute = newStreamRouteClient(c)
	c.globalRules = newGlobalRuleClient(c)
	c.consumer = newConsumerClient(c)
	c.plugin = newPluginClient(c)
	c.schema = newSchemaClient(c)
	c.pluginConfig = newPluginConfigClient(c)
	c.pluginMetadata = newPluginMetadataClient(c)
	c.validator = newDummyValidator()

	c.cache, err = cache.NewMemDBCache()
	if err != nil {
		return nil, err
	}

	if o.CacheSynced {
		c.waitforCacheSync = true
		go c.syncCache(ctx)
	}

	return c, nil
}

func (c *cluster) syncCache(ctx context.Context) {
	log.Infow("syncing cache", zap.String("cluster", c.name))
	now := time.Now()
	defer func() {
		if c.cacheSyncErr == nil {
			log.Infow("cache synced",
				zap.String("cost_time", time.Since(now).String()),
				zap.String("cluster", c.name),
			)
		} else {
			log.Errorw("failed to sync cache",
				zap.String("cost_time", time.Since(now).String()),
				zap.String("cluster", c.name),
			)
		}
	}()

	backoff := wait.Backoff{
		Duration: 2 * time.Second,
		Factor:   1,
		Steps:    5,
	}
	var lastSyncErr error
	err := wait.ExponentialBackoff(backoff, func() (done bool, err error) {
		// impossibly return: false, nil
		// so can safe used
		done, lastSyncErr = c.syncCacheOnce(ctx)
		select {
		case <-ctx.Done():
			err = context.Canceled
		default:
			break
		}
		return
	})
	if err != nil {
		// if ErrWaitTimeout then set lastSyncErr
		c.cacheSyncErr = lastSyncErr
	}
	close(c.cacheSynced)

	if !atomic.CompareAndSwapInt32(&c.cacheState, _cacheSyncing, _cacheSynced) {
		panic("dubious state when sync cache")
	}
}

func (c *cluster) syncCacheOnce(ctx context.Context) (bool, error) {
	routes, err := c.route.List(ctx)
	if err != nil {
		log.Errorf("failed to list routes in APISIX: %s", err)
		return false, err
	}
	ssl, err := c.ssl.List(ctx)
	if err != nil {
		log.Errorf("failed to list ssl in APISIX: %s", err)
		return false, err
	}
	globalRules, err := c.globalRules.List(ctx)
	if err != nil {
		log.Errorf("failed to list global_rules in APISIX: %s", err)
		return false, err
	}
	consumers, err := c.consumer.List(ctx)
	if err != nil {
		log.Errorf("failed to list consumers in APISIX: %s", err)
		return false, err
	}

	for _, r := range routes {
		if err := c.cache.InsertRoute(r); err != nil {
			log.Errorw("failed to insert route to cache",
				zap.String("route", r.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, s := range ssl {
		if err := c.cache.InsertSSL(s); err != nil {
			log.Errorw("failed to insert ssl to cache",
				zap.String("ssl", s.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, gr := range globalRules {
		if err := c.cache.InsertGlobalRule(gr); err != nil {
			log.Errorw("failed to insert global_rule to cache",
				zap.Any("global_rule", gr),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, consumer := range consumers {
		if err := c.cache.InsertConsumer(consumer); err != nil {
			log.Errorw("failed to insert consumer to cache",
				zap.Any("consumer", consumer),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
		}
	}
	// for _, u := range pluginConfigs {
	// 	if err := c.cache.InsertPluginConfig(u); err != nil {
	// 		log.Errorw("failed to insert pluginConfig to cache",
	// 			zap.String("pluginConfig", u.ID),
	// 			zap.String("cluster", c.name),
	// 			zap.String("error", err.Error()),
	// 		)
	// 		return false, err
	// 	}
	// }
	return true, nil
}

// String implements Cluster.String method.
func (c *cluster) String() string {
	return fmt.Sprintf("name=%s; base_url=%s", c.name, c.baseURL)
}

// HasSynced implements Cluster.HasSynced method.
func (c *cluster) HasSynced(ctx context.Context) error {
	if !c.waitforCacheSync {
		return nil
	}
	if c.cacheSyncErr != nil {
		return c.cacheSyncErr
	}
	if atomic.LoadInt32(&c.cacheState) == _cacheSynced {
		return nil
	}

	// still in sync
	now := time.Now()
	log.Warnf("waiting cluster %s to ready, it may takes a while", c.name)
	select {
	case <-ctx.Done():
		log.Errorf("failed to wait cluster to ready: %s", ctx.Err())
		return ctx.Err()
	case <-c.cacheSynced:
		if c.cacheSyncErr != nil {
			// See https://github.com/apache/api7-ingress-controller/issues/448
			// for more details.
			return c.cacheSyncErr
		}
		log.Warnf("cluster %s now is ready, cost time %s", c.name, time.Since(now).String())
		return nil
	}
}

// Route implements Cluster.Route method.
func (c *cluster) Route() Route {
	return c.route
}

// Upstream implements Cluster.Upstream method.
func (c *cluster) Service() Service {
	return c.service
}

// SSL implements Cluster.SSL method.
func (c *cluster) SSL() SSL {
	return c.ssl
}

// StreamRoute implements Cluster.StreamRoute method.
func (c *cluster) StreamRoute() StreamRoute {
	return c.streamRoute
}

// GlobalRule implements Cluster.GlobalRule method.
func (c *cluster) GlobalRule() GlobalRule {
	return c.globalRules
}

// Consumer implements Cluster.Consumer method.
func (c *cluster) Consumer() Consumer {
	return c.consumer
}

// Plugin implements Cluster.Plugin method.
func (c *cluster) Plugin() Plugin {
	return c.plugin
}

// PluginConfig implements Cluster.PluginConfig method.
func (c *cluster) PluginConfig() PluginConfig {
	return c.pluginConfig
}

// Schema implements Cluster.Schema method.
func (c *cluster) Schema() Schema {
	return c.schema
}

func (c *cluster) PluginMetadata() PluginMetadata {
	return c.pluginMetadata
}

func (c *cluster) Validator() APISIXSchemaValidator {
	return c.validator
}

// HealthCheck implements Cluster.HealthCheck method.
//
// It checks the health of an APISIX cluster by performing a TCP socket probe
// against the baseURLHost. It will retry up to 3 times with exponential backoff
// before returning an error.
//
// Parameters:
//
//	ctx: The context for the health check.
//
// Returns:
//
//	err: Any error encountered while performing the health check.
func (c *cluster) HealthCheck(ctx context.Context) (err error) {
	// Retry three times in a row, and exit if all of them fail.
	backoff := wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   1,
		Steps:    3,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (done bool, _ error) {
		if lastCheckErr := c.healthCheck(ctx); lastCheckErr != nil {
			log.Warnf("failed to check health for cluster %s: %s, will retry", c.name, lastCheckErr)
			return
		}
		done = true
		return
	})

	return err
}

func (c *cluster) healthCheck(ctx context.Context) (err error) {
	// tcp socket probe
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", c.baseURLHost)
	if err != nil {
		return err
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Warnw("failed to close tcp probe connection",
				zap.Error(err),
				zap.String("cluster", c.name),
			)
		}
	}(conn)

	return
}

func (c *cluster) applyAuth(req *http.Request) {
	if c.adminKey != "" {
		req.Header.Set("X-API-Key", c.adminKey)
	}
}

func (c *cluster) do(req *http.Request) (*http.Response, error) {
	c.applyAuth(req)
	return c.cli.Do(req)
}

func (c *cluster) isFunctionDisabled(body string) bool {
	return strings.Contains(body, "is disabled")
}

func (c *cluster) getResource(ctx context.Context, url, resource string) (*getResponse, error) {
	log.Debugw("get resource in cluster",
		zap.String("cluster_name", c.name),
		zap.String("name", resource),
		zap.String("url", url),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		body := readBody(resp.Body, url)
		if c.isFunctionDisabled(body) {
			return nil, ErrFunctionDisabled
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, cache.ErrNotFound
		} else {
			err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
			err = multierr.Append(err, fmt.Errorf("error message: %s", body))
		}
		return nil, err
	}

	if c.adminVersion == "v3" {
		var res getResponse

		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}
	var res getResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *cluster) listResource(ctx context.Context, url, resource string) (listResponse, error) {
	log.Debugw("list resource in cluster",
		zap.String("cluster_name", c.name),
		zap.String("name", resource),
		zap.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return listResponse{}, err
	}
	resp, err := c.do(req)
	if err != nil {
		return listResponse{}, err
	}

	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		body := readBody(resp.Body, url)
		if c.isFunctionDisabled(body) {
			return listResponse{}, ErrFunctionDisabled
		}
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", body))
		return listResponse{}, err
	}
	if c.adminVersion == "v3" {
		var list listResponse
		byt, _ := io.ReadAll(resp.Body)
		err := json.Unmarshal(byt, &list)
		if err != nil {
			return listResponse{}, err
		}
		return list, nil
	}
	var list listResponse

	byt, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(byt, &list)
	if err != nil {
		return listResponse{}, err
	}
	return list, nil
}

func (c *cluster) createResource(ctx context.Context, url, resource string, body []byte) (*getResponse, error) {
	log.Debugw("creating resource in cluster",
		zap.String("cluster_name", c.name),
		zap.String("name", resource),
		zap.String("url", url),
		zap.ByteString("body", body),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body := readBody(resp.Body, url)
		if c.isFunctionDisabled(body) {
			return nil, ErrFunctionDisabled
		}
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", body))
		return nil, err
	}

	var cr getResponse
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(byt, &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

func (c *cluster) updateResource(ctx context.Context, url, resource string, body []byte) (*getResponse, error) {
	log.Debugw("updating resource in cluster",
		zap.String("cluster_name", c.name),
		zap.String("name", resource),
		zap.String("url", url),
		zap.ByteString("body", body),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body := readBody(resp.Body, url)
		log.Debugw("update response",
			zap.Int("status code %d", resp.StatusCode),
			zap.String("body %s", body),
		)
		if c.isFunctionDisabled(body) {
			return nil, ErrFunctionDisabled
		}
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", body))
		return nil, err
	}
	if c.adminVersion == "v3" {
		var ur updateResponseV3

		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&ur); err != nil {
			return nil, err
		}

		return &ur, nil
	}
	var ur updateResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&ur); err != nil {
		return nil, err
	}
	return &ur, nil
}

func (c *cluster) deleteResource(ctx context.Context, url, resource string) error {
	log.Debugw("deleting resource in cluster",
		zap.String("cluster_name", c.name),
		zap.String("name", resource),
		zap.String("url", url),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}

	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusNotFound {
		message := readBody(resp.Body, url)
		if c.isFunctionDisabled(message) {
			return ErrFunctionDisabled
		}
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", message))
		if strings.Contains(message, "still using") {
			return cache.ErrStillInUse
		}
		return err
	}
	return nil
}

// drainBody reads whole data until EOF from r, then close it.
func drainBody(r io.ReadCloser, url string) {
	_, err := io.Copy(io.Discard, r)
	if err != nil {
		if err.Error() != _errReadOnClosedResBody.Error() {
			log.Warnw("failed to drain body (read)",
				zap.String("url", url),
				zap.Error(err),
			)
		}
	}

	if err := r.Close(); err != nil {
		log.Warnw("failed to drain body (close)",
			zap.String("url", url),
			zap.Error(err),
		)
	}
}

func readBody(r io.ReadCloser, url string) string {
	defer func() {
		if err := r.Close(); err != nil {
			log.Warnw("failed to close body", zap.String("url", url), zap.Error(err))
		}
	}()
	data, err := io.ReadAll(r)
	if err != nil {
		log.Warnw("failed to read body", zap.String("url", url), zap.Error(err))
		return ""
	}
	return string(data)
}

// getSchema returns the schema of APISIX object.
func (c *cluster) getSchema(_ context.Context, url, resource string) (string, error) {
	log.Debugw("get schema in cluster",
		zap.String("url", url),
		zap.String("cluster", c.name),
		zap.String("resource", resource),
	)
	// TODO: fixme The above passed context gets cancelled for some reason. Investigate
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}

	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", cache.ErrNotFound
		} else {
			err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
			err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		}
		return "", err
	}

	return readBody(resp.Body, url), nil
}

// getList returns a list of string.
func (c *cluster) getList(ctx context.Context, url, resource string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, cache.ErrNotFound
		} else {
			err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
			err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		}
		return nil, err
	}

	// In EE, for plugins the response is an array of string and not an object.
	// sent to /list
	if resource == "plugin" {
		byt, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var listResponse []string
		err = json.Unmarshal(byt, &listResponse)
		if err != nil {
			return nil, err
		}
		return listResponse, nil
	}
	var listResponse map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&listResponse); err != nil {
		return nil, err
	}
	res := make([]string, 0, len(listResponse))

	for name := range listResponse {
		res = append(res, name)
	}
	return res, nil
}

func (c *cluster) GetGlobalRule(ctx context.Context, baseUrl, id string) (*v1.GlobalRule, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "globalRule")
	if err != nil {
		return nil, err
	}

	globalRule, err := resp.globalRule()
	if err != nil {
		log.Errorw("failed to convert global_rule item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}

	return globalRule, nil
}

func (c *cluster) GetConsumer(ctx context.Context, baseUrl, name string) (*v1.Consumer, error) {
	url := baseUrl + "/" + name
	resp, err := c.getResource(ctx, url, "consumer")
	if err != nil {
		return nil, err
	}

	consumer, err := resp.consumer()
	if err != nil {
		log.Errorw("failed to convert consumer item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}
	return consumer, nil
}

func (c *cluster) GetPluginConfig(ctx context.Context, baseUrl, id string) (*v1.PluginConfig, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "pluginConfig")
	if err != nil {
		return nil, err
	}

	pluginConfig, err := resp.pluginConfig()
	if err != nil {
		log.Errorw("failed to convert pluginConfig item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}
	return pluginConfig, nil
}

func (c *cluster) GetRoute(ctx context.Context, baseUrl, id string) (*v1.Route, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "route")
	if err != nil {
		return nil, err
	}

	route, err := resp.route()
	if err != nil {
		log.Errorw("failed to convert route item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}
	return route, nil
}

func (c *cluster) GetStreamRoute(ctx context.Context, baseUrl, id string) (*v1.StreamRoute, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "streamRoute")
	if err != nil {
		return nil, err
	}

	streamRoute, err := resp.streamRoute()
	if err != nil {
		log.Errorw("failed to convert stream_route item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}
	return streamRoute, nil
}

func (c *cluster) GetService(ctx context.Context, baseUrl, id string) (*v1.Service, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "service")
	if err != nil {
		return nil, err
	}
	svc, err := resp.service()
	if err != nil {
		log.Errorw("failed to convert service item",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, err
	}
	return svc, nil
}

func (c *cluster) GetSSL(ctx context.Context, baseUrl, id string) (*v1.Ssl, error) {
	url := baseUrl + "/" + id
	resp, err := c.getResource(ctx, url, "ssl")
	if err != nil {
		return nil, err
	}
	ssl, err := resp.ssl()
	if err != nil {
		return nil, err
	}
	return ssl, nil
}

func getFromCacheOrAPI[T any](
	ctx context.Context,
	id string,
	url string,
	cacheGet func(string) (T, error),
	cacheInsert func(T) error,
	apiGet func(context.Context, string, string) (T, error),
) (T, error) {
	item, err := cacheGet(id)
	if err == nil {
		return item, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find in cache, will try to lookup from APISIX",
			zap.Error(err),
		)
	} else {
		log.Debugw("not found in cache, will try to lookup from APISIX",
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	item, err = apiGet(ctx, url, id)
	if err != nil {
		return item, err
	}

	if err := cacheInsert(item); err != nil {
		log.Errorf("failed to reflect create to cache: %s", err)
		return item, err
	}
	return item, nil
}

func updateResource[T any](
	ctx context.Context,
	obj T,
	url string,
	resourceType string,
	apiUpdate func(context.Context, string, string, []byte) (*getResponse, error),
	cacheInsert func(T) error,
	parseResponse func(*getResponse) (T, error),
) (T, error) {
	var val T
	body, err := json.Marshal(obj)
	if err != nil {
		return val, err
	}
	resp, err := apiUpdate(ctx, url, resourceType, body)
	if err != nil {
		return val, err
	}
	val, err = parseResponse(resp)
	if err != nil {
		return val, err
	}
	if err := cacheInsert(val); err != nil {
		log.Errorf("failed to reflect update to cache: %s", err)
		return val, err
	}
	return val, nil
}
