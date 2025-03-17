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

	var task Task = Task{
		Name:   obj.GetName(),
		Labels: label.GenLabel(obj),
	}
	var extraArgs []string

	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		extraArgs = append(extraArgs, "--include-resource-type", "service")
		log.Debugw("translating http route", zap.Any("http route", obj))
		result, err = d.translator.TranslateHTTPRoute(tctx, obj.DeepCopy())
	case *gatewayv1.Gateway:
		extraArgs = append(extraArgs, "--include-resource-type", "global_rule")
		log.Debugw("translating gateway", zap.Any("gateway", obj))
		result, err = d.translator.TranslateGateway(tctx, obj.DeepCopy())
	}
	if err != nil {
		return err
	}
	log.Debugw("translated result", zap.Any("result", result))
	if result == nil {
		return nil
	}

	resources := types.Resources{
		Services:    result.Services,
		GlobalRules: result.GlobalRules,
	}
	log.Debugw("adc resources", zap.Any("resources", resources))

	task.Resources = resources

	return d.sync(task, extraArgs...)
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	task := Task{
		Name:   obj.GetName(),
		Labels: label.GenLabel(obj),
	}

	var extraArgs []string

	switch obj.(type) {
	case *gatewayv1.HTTPRoute:
		extraArgs = append(extraArgs, "--include-resource-type", "service")
	case *gatewayv1.Gateway:
		extraArgs = append(extraArgs, "--include-resource-type", "global_rule")
	}

	return d.sync(task, extraArgs...)
}

func (d *adcClient) sync(task Task, extraArgs ...string) error {
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
		"--tls-skip-verify",
	}

	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
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

	log.Debugw("adc sync success", zap.String("taskname", task.Name))

	return nil
}
