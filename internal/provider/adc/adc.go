package adc

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	types "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controller/label"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/api7/api7-ingress-controller/internal/provider/adc/translator"
	"github.com/api7/gopkg/pkg/log"
)

type adcClient struct {
	translator *translator.Translator

	ServerAddr   string
	Token        string
	GatewayGroup string
}

type Task struct {
	Name      string
	Resources types.Resources
	Labels    map[string]string
}

func New() (provider.Provider, error) {
	gc := config.GetFirstGatewayConfig()
	return &adcClient{
		translator: &translator.Translator{},
		ServerAddr: gc.ControlPlane.Endpoints[0],
		Token:      gc.ControlPlane.AdminKey,
	}, nil
}

func (d *adcClient) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	var result *translator.TranslateResult
	var err error

	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, obj.DeepCopy())
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	resources := types.Resources{
		Services: result.Services,
	}

	return d.sync(Task{
		Name:      obj.GetName(),
		Resources: resources,
		Labels:    label.GenLabel(obj),
	})
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	return d.sync(Task{
		Name:   obj.GetName(),
		Labels: label.GenLabel(obj),
	})
}

func (d *adcClient) sync(task Task) error {
	log.Debugw("syncing task", zap.Any("task", task))

	yaml, err := yaml.Marshal(task.Resources)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "adc-task-*.yaml")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	log.Debugw("syncing resources", zap.String("file", tmpFile.Name()), zap.String("yaml", string(yaml)))

	if _, err := tmpFile.Write(yaml); err != nil {
		return err
	}
	args := []string{
		"sync",
		"-f", tmpFile.Name(),
		"--include-resource-type", "service",
		"--tls-skip-verify",
	}

	for k, v := range task.Labels {
		args = append(args, "--label-selector", k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("adc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env,
		"ADC_EXPERIMENTAL_FEATURE_FLAGS=remote-state-file,parallel-backend-request",
		"ADC_BACKEND=api7ee",
		"ADC_SERVER="+d.ServerAddr,
		"ADC_TOKEN="+d.Token,
	)

	if err := cmd.Run(); err != nil {
		log.Errorw("failed to run adc",
			zap.Error(err),
			zap.String("output", stdout.String()),
			zap.String("stderr", stderr.String()),
		)
		return err
	}
	return nil
}
