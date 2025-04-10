package adc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"

	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	types "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
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
	var (
		result        *translator.TranslateResult
		resourceTypes []string
		err           error
	)

	switch t := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "ssl", "plugin_metadata")
	case *networkingv1.Ingress:
		result, err = d.translator.TranslateIngress(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service", "ssl")
	case *v1alpha1.Consumer:
		result, err = d.translator.TranslateConsumerV1alpha1(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
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
			GlobalRules:    result.GlobalRules,
			PluginMetadata: result.PluginMetadata,
			Services:       result.Services,
			SSLs:           result.SSL,
			Consumers:      result.Consumers,
		},
		ResourceTypes: resourceTypes,
	})
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	log.Debugw("deleting object", zap.Any("object", obj))

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute:
		resourceTypes = append(resourceTypes, "service")
		labels = label.GenLabel(obj)
	case *gatewayv1.Gateway:
		// delete all resources
	case *networkingv1.Ingress:
		resourceTypes = append(resourceTypes, "service", "ssl")
		labels = label.GenLabel(obj)
	case *v1alpha1.Consumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	}

	return d.sync(Task{
		Name:          obj.GetName(),
		Labels:        labels,
		ResourceTypes: resourceTypes,
	})
}

func (d *adcClient) sync(task Task) error {
	log.Debugw("syncing resources", zap.Any("task", task))

	data, err := json.Marshal(task.Resources)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "adc-task-*.json")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	log.Debugf("generated adc file, filename: %s, json: %s\n", tmpFile.Name(), string(data))

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

	adcEnv := []string{
		"ADC_EXPERIMENTAL_FEATURE_FLAGS=remote-state-file,parallel-backend-request",
		"ADC_RUNNING_MODE=ingress",
		"ADC_BACKEND=api7ee",
		"ADC_SERVER=" + d.ServerAddr,
		"ADC_TOKEN=" + d.Token,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("adc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, adcEnv...)

	log.Debug("running adc command", zap.String("command", cmd.String()), zap.Strings("env", adcEnv))

	var result types.SyncResult
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

	output := stdout.Bytes()
	if err := json.Unmarshal(output, &result); err != nil {
		log.Errorw("failed to unmarshal adc output",
			zap.Error(err),
			zap.String("stdout", string(output)),
		)
		return errors.New("failed to unmarshal adc result: " + err.Error())
	}

	if result.FailedCount > 0 {
		log.Errorw("adc sync failed", zap.Any("result", result))
		failed := result.Failed
		return errors.New(failed[0].Reason)
	}

	log.Debugw("adc sync success", zap.Any("result", result))
	return nil
}
