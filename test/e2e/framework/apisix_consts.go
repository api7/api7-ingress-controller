package framework

import (
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

var (
	//go:embed manifests/apisix-standalone.yaml
	apisixStandaloneTemplate string
	APISIXStandaloneTpl      *template.Template
)

func init() {
	tpl, err := template.New("apisix-standalone").Funcs(sprig.TxtFuncMap()).Parse(apisixStandaloneTemplate)
	if err != nil {
		panic(err)
	}
	APISIXStandaloneTpl = tpl
}
