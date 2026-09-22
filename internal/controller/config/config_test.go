package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, DefaultLogLevel, cfg.LogLevel)
	assert.Equal(t, DefaultControllerName, cfg.ControllerName)
	assert.Equal(t, DefaultLeaderElectionID, cfg.LeaderElectionID)
	assert.Equal(t, ListenerPortMatchModeOff, cfg.ListenerPortMatchMode)
}

func TestConfigValidateListenerPortMatchMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      ListenerPortMatchMode
		expectErr bool
	}{
		{
			name:      "default auto",
			mode:      ListenerPortMatchModeAuto,
			expectErr: false,
		},
		{
			name:      "explicit",
			mode:      ListenerPortMatchModeExplicit,
			expectErr: false,
		},
		{
			name:      "off",
			mode:      ListenerPortMatchModeOff,
			expectErr: false,
		},
		{
			name:      "empty mode is allowed",
			mode:      "",
			expectErr: false,
		},
		{
			name:      "invalid mode",
			mode:      "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.ListenerPortMatchMode = tt.mode

			err := cfg.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "invalid listener_port_match_mode")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidateNamespaceSelector(t *testing.T) {
	tests := []struct {
		name      string
		selector  []string
		expectErr bool
	}{
		{name: "unset", selector: nil},
		{name: "1.x default", selector: []string{""}},
		{name: "equality", selector: []string{"team=a"}},
		{name: "set based", selector: []string{"env in (prod,staging),!legacy", "team=a"}},
		{name: "invalid", selector: []string{"team in a"}, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.NamespaceSelector = tt.selector

			err := cfg.Validate()
			if tt.expectErr {
				assert.ErrorContains(t, err, "invalid namespace_selector")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseNamespaceSelector(t *testing.T) {
	nsLabels := labels.Set{"version": "v1", "env": "prod"}

	tests := []struct {
		name    string
		entries []string
		// nil means the selector is disabled.
		matches *bool
	}{
		// Cases ported from TestMultiValueLabelsIsSubsetOf of 1.x.
		{name: "no entry", entries: nil},
		{name: "1.x default", entries: []string{""}},
		{name: "single value", entries: []string{"env=prod"}, matches: ptr.To(true)},
		{name: "values on one key are ORed", entries: []string{"env=qa", "env=prod"}, matches: ptr.To(true)},
		{name: "value mismatch", entries: []string{"env=qa"}, matches: ptr.To(false)},
		{name: "missing key", entries: []string{"env3=not"}, matches: ptr.To(false)},
		// Entries on different keys are ANDed.
		{name: "all keys match", entries: []string{"env=prod", "version=v1"}, matches: ptr.To(true)},
		{name: "one key mismatches", entries: []string{"env=prod", "version=v2"}, matches: ptr.To(false)},
		{name: "empty entry is ignored", entries: []string{"env=qa", ""}, matches: ptr.To(false)},
		// Full selector syntax on top of 1.x.
		{name: "in merges with equality", entries: []string{"env in (qa)", "env==prod"}, matches: ptr.To(true)},
		{name: "not equal", entries: []string{"env=prod", "version!=v1"}, matches: ptr.To(false)},
		{name: "does not exist", entries: []string{"!legacy"}, matches: ptr.To(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := ParseNamespaceSelector(tt.entries)
			require.NoError(t, err)
			if tt.matches == nil {
				assert.Nil(t, selector)
				return
			}
			require.NotNil(t, selector)
			assert.Equal(t, *tt.matches, selector.Matches(nsLabels), selector.String())
		})
	}
}

func TestNewConfigFromFile(t *testing.T) {
	// Create a temporary config file
	fileContent := `
log_level: debug
controller_name: test-controller
disable_gateway_api: true
namespace_selector:
- "team=a"
`
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()

	_, err = tempFile.WriteString(fileContent)
	assert.NoError(t, err)
	_ = tempFile.Close()

	cfg, err := NewConfigFromFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "test-controller", cfg.ControllerName)
	assert.Equal(t, true, cfg.DisableGatewayAPI)
	assert.Equal(t, []string{"team=a"}, cfg.NamespaceSelector)
}
