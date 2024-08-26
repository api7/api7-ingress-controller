package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

var (
	ControllerConfig = NewDefaultConfig()
)

func SetControllerConfig(cfg *Config) {
	ControllerConfig = cfg
}

// NewDefaultConfig creates a Config object which fills all config items with
// default value.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:         DefaultLogLevel,
		ControllerName:   DefaultControllerName,
		LeaderElectionID: DefaultLeaderElectionID,
		ProbeAddr:        DefaultProbeAddr,
		MetricsAddr:      DefaultMetricsAddr,
	}
}

// NewConfigFromFile creates a Config object and fills all config items according
// to the configuration file. The file can be in JSON/YAML format, which will be
// distinguished according to the file suffix.
func NewConfigFromFile(filename string) (*Config, error) {
	cfg := NewDefaultConfig()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	envVarMap := map[string]string{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envVarMap[pair[0]] = pair[1]
	}

	tpl := template.New("text").Option("missingkey=error")
	tpl, err = tpl.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration template %v", err)
	}
	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, envVarMap)
	if err != nil {
		return nil, fmt.Errorf("error execute configuration template %v", err)
	}

	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(buf.Bytes(), cfg)
	} else {
		err = json.Unmarshal(buf.Bytes(), cfg)
	}

	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if err := c.validateControlPlanes(c.ControlPlanes); err != nil {
		return fmt.Errorf("failed to validate control_planes: %w", err)
	}
	if c.ControllerName == "" {
		return fmt.Errorf("controller_name is required")
	}
	return nil
}

func (c *Config) validateControlPlanes(cpc []*ControlPlaneConfig) error {
	if len(cpc) == 0 {
		return fmt.Errorf("control_planes config is required")
	}
	for _, cp := range cpc {
		if cp.GatewayName == "" {
			return fmt.Errorf("control_planesp[].gateway_name is required")
		}
		if cp.AdminAPI.AdminKey == "" {
			return fmt.Errorf("control_planes[].admin_api.admin_key is required")
		}
		if len(cp.AdminAPI.Endpoints) == 0 {
			return fmt.Errorf("control_planes[].admin_api.endpoints is required")
		}
		cp.AdminAPI.TLSVerify = new(bool)
		*cp.AdminAPI.TLSVerify = true
	}
	return nil
}

var gatewayNameMap map[string]*ControlPlaneConfig
var gatewayNameList []string

func GetControlPlaneConfigByGatewatName(gatewatName string) *ControlPlaneConfig {
	if gatewayNameMap == nil {
		gatewayNameMap = make(map[string]*ControlPlaneConfig)
		for _, cp := range ControllerConfig.ControlPlanes {
			gatewayNameMap[cp.GatewayName] = cp
		}
	}
	return gatewayNameMap[gatewatName]
}

func ControlPlanes() []*ControlPlaneConfig {
	return ControllerConfig.ControlPlanes
}

func GatewayNameList() []string {
	if gatewayNameList == nil {
		gatewayNameList = make([]string, 0, len(ControllerConfig.ControlPlanes))
		for _, cp := range ControllerConfig.ControlPlanes {
			gatewayNameList = append(gatewayNameList, cp.GatewayName)
		}
	}
	return gatewayNameList
}
