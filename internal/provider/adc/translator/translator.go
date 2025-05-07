package translator

import (
	"github.com/go-logr/logr"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
)

type Translator struct {
	Log logr.Logger
}
type TranslateResult struct {
	Routes         []*adctypes.Route
	Services       []*adctypes.Service
	SSL            []*adctypes.SSL
	GlobalRules    adctypes.GlobalRule
	PluginMetadata adctypes.PluginMetadata
	Consumers      []*adctypes.Consumer
}
