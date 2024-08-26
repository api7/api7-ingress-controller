package label

import (
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Label map[string]string

func GenLabel(client client.Object) Label {
	label := make(Label)
	label["kind"] = client.GetObjectKind().GroupVersionKind().Kind
	label["namespace"] = client.GetNamespace()
	label["name"] = client.GetName()
	label["controller_name"] = config.ControllerConfig.ControllerName
	return label
}
