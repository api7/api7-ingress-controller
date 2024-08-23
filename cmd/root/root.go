/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package root

import (
	"fmt"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/spf13/cobra"

	// +kubebuilder:scaffold:imports

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/manager"
	"github.com/api7/api7-ingress-controller/internal/version"
)

type ControlPlanesFlag struct {
	ControlPlanes []*config.ControlPlaneConfig
}

func (f *ControlPlanesFlag) String() string {
	data, _ := yaml.Marshal(f.ControlPlanes)
	return string(data)
}

func (f *ControlPlanesFlag) Set(value string) error {
	var controlPlanes []*config.ControlPlaneConfig
	if err := yaml.Unmarshal([]byte(value), &controlPlanes); err != nil {
		return err
	}
	f.ControlPlanes = controlPlanes
	return nil
}

func (f *ControlPlanesFlag) Type() string {
	return "controlPlanes"
}

func NewRootCmd() *cobra.Command {
	root := newAPI7IngressController()
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "version for api7-ingress-controller",
		Run: func(cmd *cobra.Command, _ []string) {
			if long {
				fmt.Print(version.Long())
			} else {
				fmt.Printf("api7-ingress-controller version %s\n", version.Short())
			}
		},
	}
	cmd.PersistentFlags().BoolVar(&long, "long", false, "show long mode version information")

	return cmd

}

func newAPI7IngressController() *cobra.Command {
	cfg := config.ControllerConfig
	var configPath string

	var controlPlanesFlag ControlPlanesFlag
	cmd := &cobra.Command{
		Use:     "api7-ingress-controller [command]",
		Long:    "Yet another Ingress controller for Kubernetes using api7ee Gateway as the high performance reverse proxy.",
		Version: version.Short(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath != "" {
				c, err := config.NewConfigFromFile(configPath)
				if err != nil {
					return err
				}
				cfg = c
				config.SetControllerConfig(c)
			} else {
				cfg.ControlPlanes = controlPlanesFlag.ControlPlanes
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			logLevel, err := zapcore.ParseLevel(cfg.LogLevel)
			if err != nil {
				return err
			}

			logger := zap.New(zap.UseFlagOptions(&zap.Options{
				Development: true,
				Level:       logLevel,
			}))

			logger.Info("controller start configuration", "config", cfg)
			ctrl.SetLogger(logger)

			ctx := ctrl.LoggerInto(cmd.Context(), logger)
			return manager.Run(ctx, logger)
		},
	}

	cmd.Flags().StringVarP(
		&configPath,
		"config-path",
		"c",
		"",
		"configuration file path for api7-ingress-controller",
	)
	cmd.Flags().StringVar(&cfg.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	cmd.Flags().StringVar(&cfg.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().Var(&controlPlanesFlag, "control-planes", "Control planes configuration in YAML format")
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "The log level for api7-ingress-controller")
	cmd.Flags().StringVar(&cfg.ControllerName, "controller-name", config.DefaultControllerName, "The name of the controller")

	return cmd
}
