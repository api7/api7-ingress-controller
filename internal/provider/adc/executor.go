package adc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
)

type ADCExecutor interface {
	Execute(ctx context.Context, config adcConfig, args []string) error
}

type DefaultADCExecutor struct {
	sync.Mutex
}

func (e *DefaultADCExecutor) Execute(ctx context.Context, config adcConfig, args []string) error {
	e.Lock()
	defer e.Unlock()

	return e.unlockExecute(ctx, config, args)
}

func (e *DefaultADCExecutor) unlockExecute(ctx context.Context, config adcConfig, args []string) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	serverAddr := config.ServerAddrs[0]
	token := config.Token
	tlsVerify := config.TlsVerify
	if !tlsVerify {
		args = append(args, "--tls-skip-verify")
	}

	adcEnv := []string{
		"ADC_EXPERIMENTAL_FEATURE_FLAGS=remote-state-file,parallel-backend-request",
		"ADC_RUNNING_MODE=ingress",
		"ADC_BACKEND=apisix-standalone",
		"ADC_SERVER=" + serverAddr,
		"ADC_TOKEN=" + token,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctxWithTimeout, "adc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, adcEnv...)

	log.Debug("running adc command", zap.String("command", cmd.String()), zap.Strings("env", adcEnv))

	var result adctypes.SyncResult
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
