# BackendTrafficPolicy Health Check Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add active and passive upstream health check configuration support to `BackendTrafficPolicy` (Gateway API path), mirroring the capability already available in `ApisixUpstream`.

**Architecture:** Define health check types independently in `api/v1alpha1` package; extend `BackendTrafficPolicySpec`; add translation logic in `internal/adc/translator/policies.go` that maps v1alpha1 types to ADC types (reusing patterns from `apisixupstream.go`).

**Tech Stack:** Go, controller-gen (for DeepCopy generation), Kubernetes API conventions, APISIX ADC types.

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `api/v1alpha1/backendtrafficpolicy_types.go` | Modify | Add `HealthCheck` field + new health check types |
| `api/v1alpha1/zz_generated.deepcopy.go` | Regenerate | Auto-generated DeepCopy for new types |
| `internal/adc/translator/policies.go` | Modify | Translate health check spec → ADC upstream |
| `internal/adc/translator/httproute_test.go` | Modify | Add unit tests for health check translation |

---

### Task 1: Add Health Check Types and Field to BackendTrafficPolicySpec

**Files:**
- Modify: `api/v1alpha1/backendtrafficpolicy_types.go`

- [ ] **Step 1: Add `HealthCheck` field to `BackendTrafficPolicySpec`**

In `api/v1alpha1/backendtrafficpolicy_types.go`, add the field after `Host`:

```go
// HealthCheck defines active and passive health check configuration for
// the upstream backends. When configured, APISIX will probe backends
// (active) or monitor live traffic (passive) to detect and bypass
// unhealthy nodes.
// +optional
HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
```

- [ ] **Step 2: Add health check type definitions**

Append the following types at the end of `api/v1alpha1/backendtrafficpolicy_types.go` (before the `func init()` — but there is no init, so just append after the last type):

```go
// HealthCheck defines the active and passive health check configuration for upstream nodes.
type HealthCheck struct {
	// Active health checks proactively send requests to upstream nodes to determine their availability.
	// +kubebuilder:validation:Required
	Active *ActiveHealthCheck `json:"active" yaml:"active"`
	// Passive health checks evaluate upstream health based on observed traffic (timeouts, errors).
	// +kubebuilder:validation:Optional
	Passive *PassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// ActiveHealthCheck defines the active upstream health check configuration.
type ActiveHealthCheck struct {
	// Type is the health check type. Can be `http`, `https`, or `tcp`.
	// +kubebuilder:validation:Enum=http;https;tcp;
	// +kubebuilder:default=http
	// +optional
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Timeout sets health check timeout.
	// +optional
	Timeout metav1.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Concurrency sets the number of targets to be checked at the same time.
	// +kubebuilder:validation:Minimum=0
	// +optional
	Concurrency int `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`

	// Host sets the upstream host used in the health check request.
	// +optional
	Host string `json:"host,omitempty" yaml:"host,omitempty"`

	// Port sets the port on the upstream node to probe.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	// HTTPPath sets the HTTP path for the probe request.
	// +optional
	HTTPPath string `json:"httpPath,omitempty" yaml:"httpPath,omitempty"`

	// StrictTLS controls whether TLS certificate validation is enforced.
	// +optional
	StrictTLS *bool `json:"strictTLS,omitempty" yaml:"strictTLS,omitempty"`

	// RequestHeaders sets additional HTTP request headers for the probe.
	// +optional
	RequestHeaders []string `json:"requestHeaders,omitempty" yaml:"requestHeaders,omitempty"`

	// Healthy configures the thresholds for marking a node healthy.
	// +optional
	Healthy *ActiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`

	// Unhealthy configures the thresholds for marking a node unhealthy.
	// +optional
	Unhealthy *ActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// PassiveHealthCheck defines passive health check configuration based on observed traffic.
type PassiveHealthCheck struct {
	// Type is the passive health check type. Can be `http`, `https`, or `tcp`.
	// +kubebuilder:validation:Enum=http;https;tcp;
	// +kubebuilder:default=http
	// +optional
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Healthy defines conditions under which a node is considered healthy.
	// +optional
	Healthy *PassiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`

	// Unhealthy defines conditions under which a node is considered unhealthy.
	// +optional
	Unhealthy *PassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// ActiveHealthCheckHealthy defines thresholds for actively marking an upstream node healthy.
type ActiveHealthCheckHealthy struct {
	PassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// Interval defines the time between health check probes.
	// Minimum is 1s.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// ActiveHealthCheckUnhealthy defines thresholds for actively marking an upstream node unhealthy.
type ActiveHealthCheckUnhealthy struct {
	PassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// Interval defines the time between health check probes.
	// Minimum is 1s.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// PassiveHealthCheckHealthy defines conditions for passively marking a node healthy.
type PassiveHealthCheckHealthy struct {
	// HTTPCodes is the list of HTTP status codes considered healthy.
	// +kubebuilder:validation:MinItems=1
	// +optional
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`

	// Successes is the number of consecutive successful responses required to mark a node healthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	Successes int `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// PassiveHealthCheckUnhealthy defines conditions for passively marking a node unhealthy.
type PassiveHealthCheckUnhealthy struct {
	// HTTPCodes is the list of HTTP status codes considered unhealthy.
	// +kubebuilder:validation:MinItems=1
	// +optional
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`

	// HTTPFailures is the number of HTTP failures to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	HTTPFailures int `json:"httpFailures,omitempty" yaml:"httpFailures,omitempty"`

	// TCPFailures is the number of TCP failures to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	TCPFailures int `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`

	// Timeouts is the number of timeouts to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=254
	// +optional
	Timeouts int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}
```

- [ ] **Step 3: Build to verify no syntax errors**

```bash
go build ./api/v1alpha1/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add api/v1alpha1/backendtrafficpolicy_types.go
git commit -m "feat(api): add HealthCheck types to BackendTrafficPolicySpec"
```

---

### Task 2: Regenerate DeepCopy Methods

**Files:**
- Modify: `api/v1alpha1/zz_generated.deepcopy.go` (auto-generated)

- [ ] **Step 1: Run controller-gen to regenerate DeepCopy**

```bash
make generate
```

Expected: `api/v1alpha1/zz_generated.deepcopy.go` is updated with DeepCopy methods for `HealthCheck`, `ActiveHealthCheck`, `PassiveHealthCheck`, `ActiveHealthCheckHealthy`, `ActiveHealthCheckUnhealthy`, `PassiveHealthCheckHealthy`, `PassiveHealthCheckUnhealthy`.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go
git commit -m "chore: regenerate deepcopy for BackendTrafficPolicy health check types"
```

---

### Task 3: Write Failing Tests for Health Check Translation

**Files:**
- Modify: `internal/adc/translator/httproute_test.go`

- [ ] **Step 1: Add test for active health check translation**

Add the following test function to `internal/adc/translator/httproute_test.go`:

```go
func TestAttachBackendTrafficPolicyHealthCheck(t *testing.T) {
	trueVal := true

	tests := []struct {
		name       string
		policy     *v1alpha1.BackendTrafficPolicy
		wantChecks *adctypes.UpstreamHealthCheck
	}{
		{
			name:       "nil health check produces no checks",
			policy:     &v1alpha1.BackendTrafficPolicy{},
			wantChecks: nil,
		},
		{
			name: "active health check only",
			policy: &v1alpha1.BackendTrafficPolicy{
				Spec: v1alpha1.BackendTrafficPolicySpec{
					HealthCheck: &v1alpha1.HealthCheck{
						Active: &v1alpha1.ActiveHealthCheck{
							Type:        "http",
							HTTPPath:    "/healthz",
							Concurrency: 10,
							Host:        "example.com",
							Port:        8080,
							StrictTLS:   &trueVal,
							Healthy: &v1alpha1.ActiveHealthCheckHealthy{
								Interval: metav1.Duration{Duration: 5 * time.Second},
								PassiveHealthCheckHealthy: v1alpha1.PassiveHealthCheckHealthy{
									HTTPCodes: []int{200, 201},
									Successes: 3,
								},
							},
							Unhealthy: &v1alpha1.ActiveHealthCheckUnhealthy{
								Interval: metav1.Duration{Duration: 2 * time.Second},
								PassiveHealthCheckUnhealthy: v1alpha1.PassiveHealthCheckUnhealthy{
									HTTPCodes:    []int{500, 503},
									HTTPFailures: 5,
									TCPFailures:  2,
									Timeouts:     3,
								},
							},
						},
					},
				},
			},
			wantChecks: &adctypes.UpstreamHealthCheck{
				Active: &adctypes.UpstreamActiveHealthCheck{
					Type:                   "http",
					HTTPPath:               "/healthz",
					Concurrency:            10,
					Host:                   "example.com",
					Port:                   8080,
					HTTPSVerifyCertificate: true,
					Healthy: adctypes.UpstreamActiveHealthCheckHealthy{
						Interval: 5,
						UpstreamPassiveHealthCheckHealthy: adctypes.UpstreamPassiveHealthCheckHealthy{
							HTTPStatuses: []int{200, 201},
							Successes:    3,
						},
					},
					Unhealthy: adctypes.UpstreamActiveHealthCheckUnhealthy{
						Interval: 2,
						UpstreamPassiveHealthCheckUnhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
							HTTPStatuses: []int{500, 503},
							HTTPFailures: 5,
							TCPFailures:  2,
							Timeouts:     3,
						},
					},
				},
			},
		},
		{
			name: "passive health check only",
			policy: &v1alpha1.BackendTrafficPolicy{
				Spec: v1alpha1.BackendTrafficPolicySpec{
					HealthCheck: &v1alpha1.HealthCheck{
						Active: &v1alpha1.ActiveHealthCheck{
							Healthy: &v1alpha1.ActiveHealthCheckHealthy{
								Interval: metav1.Duration{Duration: 1 * time.Second},
							},
						},
						Passive: &v1alpha1.PassiveHealthCheck{
							Type: "http",
							Healthy: &v1alpha1.PassiveHealthCheckHealthy{
								HTTPCodes: []int{200},
								Successes: 2,
							},
							Unhealthy: &v1alpha1.PassiveHealthCheckUnhealthy{
								HTTPCodes:    []int{500},
								HTTPFailures: 3,
							},
						},
					},
				},
			},
			wantChecks: &adctypes.UpstreamHealthCheck{
				Active: &adctypes.UpstreamActiveHealthCheck{
					Type:                   "http",
					HTTPSVerifyCertificate: true,
					Healthy: adctypes.UpstreamActiveHealthCheckHealthy{
						Interval: 1,
					},
				},
				Passive: &adctypes.UpstreamPassiveHealthCheck{
					Type: "http",
					Healthy: adctypes.UpstreamPassiveHealthCheckHealthy{
						HTTPStatuses: []int{200},
						Successes:    2,
					},
					Unhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
						HTTPStatuses: []int{500},
						HTTPFailures: 3,
					},
				},
			},
		},
	}

	translator := &Translator{Log: logr.Discard()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ups := adctypes.NewDefaultUpstream()
			translator.attachBackendTrafficPolicyToUpstream(tt.policy, ups)
			assert.Equal(t, tt.wantChecks, ups.Checks)
		})
	}
}
```

Note: you need to add the following imports to `httproute_test.go` if not already present:
- `"time"`
- `adctypes "github.com/apache/apisix-ingress-controller/api/adc"`

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/adc/translator/... -run TestAttachBackendTrafficPolicyHealthCheck -v
```

Expected: compilation error or test failure (health check field not yet translated).

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/adc/translator/httproute_test.go
git commit -m "test: add failing tests for BackendTrafficPolicy health check translation"
```

---

### Task 4: Implement Health Check Translation in policies.go

**Files:**
- Modify: `internal/adc/translator/policies.go`

- [ ] **Step 1: Add health check translation to `attachBackendTrafficPolicyToUpstream`**

In `internal/adc/translator/policies.go`, update the `attachBackendTrafficPolicyToUpstream` function to call a new helper, and add the helper functions. The full updated file content:

```go
func (t *Translator) attachBackendTrafficPolicyToUpstream(policy *v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if policy == nil {
		return
	}
	upstream.PassHost = policy.Spec.PassHost
	upstream.UpstreamHost = string(policy.Spec.Host)
	upstream.Scheme = policy.Spec.Scheme
	if policy.Spec.Retries != nil {
		upstream.Retries = new(int64)
		*upstream.Retries = int64(*policy.Spec.Retries)
	}
	if policy.Spec.Timeout != nil {
		upstream.Timeout = &adctypes.Timeout{
			Connect: int(policy.Spec.Timeout.Connect.Seconds()),
			Read:    int(policy.Spec.Timeout.Read.Seconds()),
			Send:    int(policy.Spec.Timeout.Send.Seconds()),
		}
	}
	if policy.Spec.LoadBalancer != nil {
		upstream.Type = adctypes.UpstreamType(policy.Spec.LoadBalancer.Type)
		upstream.HashOn = policy.Spec.LoadBalancer.HashOn
		upstream.Key = policy.Spec.LoadBalancer.Key
	}
	if policy.Spec.HealthCheck != nil {
		upstream.Checks = translateBTPHealthCheck(policy.Spec.HealthCheck)
	}
}

func translateBTPHealthCheck(hc *v1alpha1.HealthCheck) *adctypes.UpstreamHealthCheck {
	if hc == nil || (hc.Active == nil && hc.Passive == nil) {
		return nil
	}
	result := &adctypes.UpstreamHealthCheck{}
	if hc.Active != nil {
		result.Active = translateBTPActiveHealthCheck(hc.Active)
	}
	if hc.Passive != nil {
		result.Passive = translateBTPPassiveHealthCheck(hc.Passive)
	}
	return result
}

func translateBTPActiveHealthCheck(config *v1alpha1.ActiveHealthCheck) *adctypes.UpstreamActiveHealthCheck {
	active := &adctypes.UpstreamActiveHealthCheck{
		Type:        config.Type,
		Timeout:     int(config.Timeout.Seconds()),
		Concurrency: config.Concurrency,
		Host:        config.Host,
		Port:        config.Port,
		HTTPPath:    config.HTTPPath,
	}
	if config.Type == "" {
		active.Type = "http"
	}
	if config.StrictTLS == nil || *config.StrictTLS {
		active.HTTPSVerifyCertificate = true
	}
	if len(config.RequestHeaders) > 0 {
		active.HTTPRequestHeaders = config.RequestHeaders
	}
	if config.Healthy != nil {
		active.Healthy = adctypes.UpstreamActiveHealthCheckHealthy{
			Interval: int(config.Healthy.Interval.Seconds()),
			UpstreamPassiveHealthCheckHealthy: adctypes.UpstreamPassiveHealthCheckHealthy{
				HTTPStatuses: config.Healthy.HTTPCodes,
				Successes:    config.Healthy.Successes,
			},
		}
	}
	if config.Unhealthy != nil {
		active.Unhealthy = adctypes.UpstreamActiveHealthCheckUnhealthy{
			Interval: int(config.Unhealthy.Interval.Seconds()),
			UpstreamPassiveHealthCheckUnhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
				HTTPStatuses: config.Unhealthy.HTTPCodes,
				HTTPFailures: config.Unhealthy.HTTPFailures,
				TCPFailures:  config.Unhealthy.TCPFailures,
				Timeouts:     config.Unhealthy.Timeouts,
			},
		}
	}
	return active
}

func translateBTPPassiveHealthCheck(config *v1alpha1.PassiveHealthCheck) *adctypes.UpstreamPassiveHealthCheck {
	passive := &adctypes.UpstreamPassiveHealthCheck{
		Type: config.Type,
	}
	if config.Type == "" {
		passive.Type = "http"
	}
	if config.Healthy != nil {
		passive.Healthy = adctypes.UpstreamPassiveHealthCheckHealthy{
			HTTPStatuses: config.Healthy.HTTPCodes,
			Successes:    config.Healthy.Successes,
		}
	}
	if config.Unhealthy != nil {
		passive.Unhealthy = adctypes.UpstreamPassiveHealthCheckUnhealthy{
			HTTPStatuses: config.Unhealthy.HTTPCodes,
			HTTPFailures: config.Unhealthy.HTTPFailures,
			TCPFailures:  config.Unhealthy.TCPFailures,
			Timeouts:     config.Unhealthy.Timeouts,
		}
	}
	return passive
}
```

- [ ] **Step 2: Run the tests to verify they pass**

```bash
go test ./internal/adc/translator/... -run TestAttachBackendTrafficPolicyHealthCheck -v
```

Expected: all test cases PASS.

- [ ] **Step 3: Run the full translator test suite**

```bash
go test ./internal/adc/translator/... -v
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/adc/translator/policies.go
git commit -m "feat: translate BackendTrafficPolicy health check to APISIX upstream"
```

---

### Task 5: Verify Build and Full Test Suite

- [ ] **Step 1: Run full build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 2: Run all unit tests**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 3: Regenerate manifests (CRD YAML)**

```bash
make manifests
```

Expected: CRD YAML in `config/crd/bases/` is updated to include `healthCheck` fields in `BackendTrafficPolicy`.

- [ ] **Step 4: Commit CRD changes**

```bash
git add config/crd/bases/
git commit -m "chore: regenerate CRD manifests with BackendTrafficPolicy health check fields"
```

---

## Notes on ADC Type Field Names

When writing translation code, the ADC types (`api/adc/types.go`) use these field names:

- `UpstreamHealthCheck.Active` → `*UpstreamActiveHealthCheck`
- `UpstreamHealthCheck.Passive` → `*UpstreamPassiveHealthCheck`
- `UpstreamActiveHealthCheck.Healthy` → `UpstreamActiveHealthCheckHealthy` (value, not pointer)
- `UpstreamActiveHealthCheck.Unhealthy` → `UpstreamActiveHealthCheckUnhealthy` (value, not pointer)
- `UpstreamActiveHealthCheckHealthy` embeds `UpstreamPassiveHealthCheckHealthy` inline
- `UpstreamPassiveHealthCheckHealthy.HTTPStatuses` (not HTTPCodes)
- `UpstreamPassiveHealthCheckUnhealthy.HTTPStatuses` (not HTTPCodes)
- `Upstream.Checks` is the field for `*UpstreamHealthCheck`

Verify exact field names against `api/adc/types.go` before compiling.
