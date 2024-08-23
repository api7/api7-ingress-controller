package framework

import (
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

var (
	//go:embed manifests/ingress.yaml
	_ingressSpec   string
	IngressSpecTpl *template.Template
)

func init() {
	tpl, err := template.New("ingress").Funcs(sprig.TxtFuncMap()).Parse(_ingressSpec)
	if err != nil {
		panic(err)
	}
	IngressSpecTpl = tpl
}
