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

package scaffold

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/provider/adc/translator"
)

// DataplaneResource defines the interface for accessing dataplane resources
type DataplaneResource interface {
	Route() RouteResource
	Service() ServiceResource
	SSL() SSLResource
	Consumer() ConsumerResource
}

// RouteResource defines the interface for route resources
type RouteResource interface {
	List(ctx context.Context) ([]*adctypes.Route, error)
}

// ServiceResource defines the interface for service resources
type ServiceResource interface {
	List(ctx context.Context) ([]*adctypes.Service, error)
}

// SSLResource defines the interface for SSL resources
type SSLResource interface {
	List(ctx context.Context) ([]*adctypes.SSL, error)
}

// ConsumerResource defines the interface for consumer resources
type ConsumerResource interface {
	List(ctx context.Context) ([]*adctypes.Consumer, error)
}

// adcDataplaneResource implements DataplaneResource using ADC command
type adcDataplaneResource struct {
	backend     string
	serverAddr  string
	token       string
	tlsVerify   bool
	syncTimeout time.Duration
}

// newADCDataplaneResource creates a new ADC-based dataplane resource
func newADCDataplaneResource(backend, serverAddr, token string, tlsVerify bool) DataplaneResource {
	return &adcDataplaneResource{
		backend:     backend,
		serverAddr:  serverAddr,
		token:       token,
		tlsVerify:   tlsVerify,
		syncTimeout: 30 * time.Second,
	}
}

func (a *adcDataplaneResource) Route() RouteResource {
	return &adcRouteResource{a}
}

func (a *adcDataplaneResource) Service() ServiceResource {
	return &adcServiceResource{a}
}

func (a *adcDataplaneResource) SSL() SSLResource {
	return &adcSSLResource{a}
}

func (a *adcDataplaneResource) Consumer() ConsumerResource {
	return &adcConsumerResource{a}
}

// dumpResources executes adc dump command and returns the resources
func (a *adcDataplaneResource) dumpResources(ctx context.Context) (*translator.TranslateResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, a.syncTimeout)
	defer cancel()

	args := []string{"dump", "-o", "/tmp/dump.yaml"}
	if !a.tlsVerify {
		args = append(args, "--tls-skip-verify")
	}

	adcEnv := []string{
		"ADC_BACKEND=" + a.backend,
		"ADC_SERVER=" + a.serverAddr,
		"ADC_TOKEN=" + a.token,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctxWithTimeout, "adc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, adcEnv...)

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var resources adctypes.Resources
	if err := json.Unmarshal(stdout.Bytes(), &resources); err != nil {
		return nil, err
	}

	// Extract routes from services
	var routes []*adctypes.Route
	for _, service := range resources.Services {
		routes = append(routes, service.Routes...)
	}

	return &translator.TranslateResult{
		Routes:         routes,
		Services:       resources.Services,
		SSL:            resources.SSLs,
		GlobalRules:    resources.GlobalRules,
		PluginMetadata: resources.PluginMetadata,
		Consumers:      resources.Consumers,
	}, nil
}

// adcRouteResource implements RouteResource
type adcRouteResource struct {
	*adcDataplaneResource
}

func (r *adcRouteResource) List(ctx context.Context) ([]*adctypes.Route, error) {
	result, err := r.dumpResources(ctx)
	if err != nil {
		return nil, err
	}
	return result.Routes, nil
}

// adcServiceResource implements ServiceResource
type adcServiceResource struct {
	*adcDataplaneResource
}

func (s *adcServiceResource) List(ctx context.Context) ([]*adctypes.Service, error) {
	result, err := s.dumpResources(ctx)
	if err != nil {
		return nil, err
	}
	return result.Services, nil
}

// adcSSLResource implements SSLResource
type adcSSLResource struct {
	*adcDataplaneResource
}

func (s *adcSSLResource) List(ctx context.Context) ([]*adctypes.SSL, error) {
	result, err := s.dumpResources(ctx)
	if err != nil {
		return nil, err
	}
	return result.SSL, nil
}

// adcConsumerResource implements ConsumerResource
type adcConsumerResource struct {
	*adcDataplaneResource
}

func (c *adcConsumerResource) List(ctx context.Context) ([]*adctypes.Consumer, error) {
	result, err := c.dumpResources(ctx)
	if err != nil {
		return nil, err
	}
	return result.Consumers, nil
}
