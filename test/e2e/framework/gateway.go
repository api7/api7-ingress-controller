package framework

import (
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

var (
	//go:embed manifests/dp.yaml
	_dpSpec   string
	DPSpecTpl *template.Template
)

func init() {
	tpl, err := template.New("dp").Funcs(sprig.TxtFuncMap()).Parse(_dpSpec)
	if err != nil {
		panic(err)
	}
	DPSpecTpl = tpl
}
