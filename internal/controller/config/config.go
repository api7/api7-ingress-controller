package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/api7/api7-ingress-controller/internal/types"
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
		LeaderElection:   NewLeaderElection(),
	}
}

func NewLeaderElection() *LeaderElection {
	return &LeaderElection{
		LeaseDuration: types.TimeDuration{Duration: 15 * time.Second},
		RenewDeadline: types.TimeDuration{Duration: 10 * time.Second},
		RetryPeriod:   types.TimeDuration{Duration: 2 * time.Second},
		Disable:       false,
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
	if c.ControllerName == "" {
		return fmt.Errorf("controller_name is required")
	}
	return nil
}

func GetControllerName() string {
	return ControllerConfig.ControllerName
}
