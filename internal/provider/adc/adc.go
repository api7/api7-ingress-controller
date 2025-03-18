package adc

import (
	"bytes"
	"context"
	"errors"
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
	Name          string
	Resources     types.Resources
	Labels        map[string]string
	ResourceTypes []string
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
	log.Debugw("updating object", zap.Any("object", obj))

	var result *translator.TranslateResult
	var resourceTypes []string
	var err error

	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, obj.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, obj.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "ssl")
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	return d.sync(Task{
		Name:   obj.GetName(),
		Labels: label.GenLabel(obj),
		Resources: types.Resources{
			Services:    result.Services,
			SSLs:        result.SSL,
			GlobalRules: result.GlobalRules,
		},
		ResourceTypes: resourceTypes,
	})
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	log.Debugw("deleting object", zap.Any("object", obj))

	resourceTypes := []string{}
	switch obj.(type) {
	case *gatewayv1.HTTPRoute:
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		resourceTypes = append(resourceTypes, "global_rule", "ssl")
	}

	return d.sync(Task{
		Name:          obj.GetName(),
		Labels:        label.GenLabel(obj),
		ResourceTypes: resourceTypes,
	})
}

func (d *adcClient) sync(task Task) error {
	log.Debugw("syncing resources", zap.Any("task", task))

	data, err := yaml.Marshal(task.Resources)
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

	log.Debugw("generated adc yaml", zap.String("file", tmpFile.Name()), zap.String("yaml", string(data)))

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	args := []string{
		"sync",
		"-f", tmpFile.Name(),
		"--tls-skip-verify",
	}

	for k, v := range task.Labels {
		args = append(args, "--label-selector", k+"="+v)
	}
	for _, t := range task.ResourceTypes {
		args = append(args, "--include-resource-type", t)
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
		stderrStr := stderr.String()
		stdoutStr := stdout.String()
		errMsg := stderrStr
		if errMsg == "" {
			errMsg = stdoutStr
		}
		log.Errorw("failed to run adc",
			zap.Error(err),
			zap.String("output", stdoutStr),
			zap.String("stderr", stderrStr),
		)
		return errors.New("failed to sync resources: " + errMsg + ", exit err: " + err.Error())
	}

	log.Debugw("adc sync success", zap.String("taskname", task.Name))
	return nil
}
