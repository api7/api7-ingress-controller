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
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(fileContent)
	assert.NoError(t, err)
	tempFile.Close()

	cfg, err := NewConfigFromFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "test-controller", cfg.ControllerName)
	assert.Equal(t, true, cfg.DisableGatewayAPI)
}
