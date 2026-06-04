package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, DefaultLogLevel, cfg.LogLevel)
	assert.Equal(t, DefaultControllerName, cfg.ControllerName)
	assert.Equal(t, DefaultLeaderElectionID, cfg.LeaderElectionID)
	assert.Equal(t, ListenerPortMatchModeAuto, cfg.ListenerPortMatchMode)
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

func TestNewConfigFromFile(t *testing.T) {
	// Create a temporary config file
	fileContent := `
log_level: debug
controller_name: test-controller
disable_gateway_api: true
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
}
