package translator

import (
	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/go-logr/logr"
)

type Translator struct {
	Log logr.Logger
}
type TranslateResult struct {
	Routes         []*adctypes.Route
	Services       []*adctypes.Service
	SSL            []*adctypes.SSL
	GlobalRules    adctypes.Plugins
	PluginMetadata adctypes.Plugins
	Consumers      []*adctypes.Consumer
}
