package adc

import (
	"io"
	"os"
	"os/exec"
	"strconv"
)

type Syncer interface {
	Run() error
	Stdout(writer io.Writer) Syncer
	Stderr(writer io.Writer) Syncer
	Verbose(level int) Syncer
	Backend(backend string) Syncer
	Server(server string) Syncer
	Token(token string) Syncer
	GatewayGroup(gatewayGroup string) Syncer
	LabelSelectors(values map[string]string) Syncer
	LabelSelector(key, value string) Syncer
	IncludeResourceTypes(resourceType ...string) Syncer
	ExcludeResourceTypes(resourceType ...string) Syncer
	Timeout(timeout string) Syncer
	CaCert(filename string) Syncer
	TLSClientCert(filename string) Syncer
	TLSClientKey(filename string) Syncer
	TLSSkipVerify() Syncer
	F(filename string) Syncer
	NoLint() Syncer
	Env(key, value string) Syncer
	Envs(kv ...string) Syncer
}

func Sync() Syncer {
	return &adcSyncer{
		args:   []string{"adc"},
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

type adcSyncer struct {
	args   []string
	envs   []string
	stdout io.Writer
	stderr io.Writer
}

func (adc *adcSyncer) Run() error {
	name, err := exec.LookPath("adc")
	if err != nil {
		name = "/bin/adc"
	}
	cmd := exec.Command(name, adc.args...)
	cmd.Env = adc.envs
	cmd.Stdout = adc.stdout
	cmd.Stderr = adc.stderr
	return cmd.Run()
}

func (adc *adcSyncer) Stdout(stdout io.Writer) Syncer {
	adc.stdout = stdout
	return adc
}

func (adc *adcSyncer) Stderr(stderr io.Writer) Syncer {
	adc.stderr = stderr
	return adc
}

func (adc *adcSyncer) Verbose(level int) Syncer {
	return adc.appendArgs("--verbose", strconv.Itoa(level))
}

func (adc *adcSyncer) Backend(backend string) Syncer {
	return adc.appendArgs("--backend", backend)
}

func (adc *adcSyncer) Server(server string) Syncer {
	return adc.appendArgs("--server", server)
}

func (adc *adcSyncer) Token(token string) Syncer {
	return adc.appendArgs("--token", token)
}

func (adc *adcSyncer) GatewayGroup(gatewayGroup string) Syncer {
	return adc.appendArgs("--gateway-group", gatewayGroup)
}

func (adc *adcSyncer) LabelSelectors(values map[string]string) Syncer {
	if len(values) > 0 {
		for key, value := range values {
			adc.LabelSelector(key, value)
		}
	}
	return adc
}

func (adc *adcSyncer) LabelSelector(key, value string) Syncer {
	return adc.appendArgs("--label-selector", key+"="+value)
}

func (adc *adcSyncer) IncludeResourceTypes(resourceType ...string) Syncer {
	for _, item := range resourceType {
		adc.appendArgs("--include-resource-type", item)
	}
	return adc
}

func (adc *adcSyncer) ExcludeResourceTypes(resourceType ...string) Syncer {
	for _, item := range resourceType {
		adc.appendArgs("--exclude-resource-type", item)
	}
	return adc
}

func (adc *adcSyncer) Timeout(timeout string) Syncer {
	return adc.appendArgs("--timeout", timeout)
}

func (adc *adcSyncer) CaCert(filename string) Syncer {
	return adc.appendArgs("--ca-cert-file", filename)
}

func (adc *adcSyncer) TLSClientCert(filename string) Syncer {
	return adc.appendArgs("--tls-client-cert-file", filename)
}

func (adc *adcSyncer) TLSClientKey(filename string) Syncer {
	return adc.appendArgs("--tls-client-key-file", filename)
}

func (adc *adcSyncer) TLSSkipVerify() Syncer {
	return adc.appendArgs("--tls-skip-verify")
}

func (adc *adcSyncer) F(filename string) Syncer {
	return adc.appendArgs("-f", filename)
}

func (adc *adcSyncer) NoLint() Syncer {
	return adc.appendArgs("--no-lint")
}

func (adc *adcSyncer) Env(key, value string) Syncer {
	return adc.Envs(key + "=" + value)
}

func (adc *adcSyncer) Envs(kv ...string) Syncer {
	adc.envs = append(adc.envs, kv...)
	return adc
}

func (adc *adcSyncer) appendArgs(args ...string) Syncer {
	adc.args = append(adc.args, args...)
	return adc
}
