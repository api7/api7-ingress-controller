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
		IngressClass:     DefaultIngressClass,
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

	if len(c.GatewayConfigs) == 0 {
		return fmt.Errorf("gateway_configs config is required")
	}
	for _, gc := range c.GatewayConfigs {
		if err := c.validateGatewayConfig(gc); err != nil {
			return fmt.Errorf("failed to validate control_planes: %w", err)
		}
	}
	if c.ControllerName == "" {
		return fmt.Errorf("controller_name is required")
	}
	return nil
}

func (c *Config) validateGatewayConfig(gc *GatewayConfig) error {

	if gc.Name == "" {
		return fmt.Errorf("control_planesp[].gateway_name is required")
	}
	if gc.ControlPlane.AdminKey == "" {
		return fmt.Errorf("control_planes[].admin_api.admin_key is required")
	}
	if len(gc.ControlPlane.Endpoints) == 0 {
		return fmt.Errorf("control_planes[].admin_api.endpoints is required")
	}
	if gc.ControlPlane.TLSVerify == nil {
		gc.ControlPlane.TLSVerify = new(bool)
		*gc.ControlPlane.TLSVerify = true
	}

	return nil
}

var gatewayNameMap map[string]*GatewayConfig
var gatewayNameList []string

func initGatewayNameMap() {
	if gatewayNameMap == nil {
		gatewayNameMap = make(map[string]*GatewayConfig)
		for _, gc := range ControllerConfig.GatewayConfigs {
			gatewayNameMap[gc.Name] = gc
		}
	}
}

func GetControlPlaneConfigByGatewatName(gatewatName string) *ControlPlaneConfig {
	initGatewayNameMap()
	if gc, ok := gatewayNameMap[gatewatName]; ok {
		return gc.ControlPlane
	}
	return nil
}

func GetGatewayConfig(gatewayName string) *GatewayConfig {
	initGatewayNameMap()
	if gc, ok := gatewayNameMap[gatewayName]; ok {
		return gc
	}
	return nil
}

func GetFirstGatewayConfig() *GatewayConfig {
	if len(ControllerConfig.GatewayConfigs) > 0 {
		return ControllerConfig.GatewayConfigs[0]
	}
	return nil
}

func GetGatewayAddresses(gatewayName string) []string {
	initGatewayNameMap()
	if gc, ok := gatewayNameMap[gatewayName]; ok {
		return gc.Addresses
	}
	return nil
}

func GatewayConfigs() []*GatewayConfig {
	return ControllerConfig.GatewayConfigs
}

func GatewayNameList() []string {
	if gatewayNameList == nil {
		gatewayNameList = make([]string, 0, len(ControllerConfig.GatewayConfigs))
		for _, gc := range ControllerConfig.GatewayConfigs {
			gatewayNameList = append(gatewayNameList, gc.Name)
		}
	}
	return gatewayNameList
}

func GetIngressClass() string {
	return ControllerConfig.IngressClass
}

func GetIngressPublishService() string {
	return ControllerConfig.IngressPublishService
}

func GetIngressStatusAddress() []string {
	return ControllerConfig.IngressStatusAddress
}

func GetControllerName() string {
	return ControllerConfig.ControllerName
}
