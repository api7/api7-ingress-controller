package label

import (
	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Label map[string]string

const (
	LabelKind           = "k8s/kind"
	LabelName           = "k8s/name"
	LabelNamespace      = "k8s/namespace"
	LabelControllerName = "k8s/controller-name"
	LabelManagedBy      = "manager-by"
)

func GenLabel(client client.Object, args ...string) Label {
	label := make(Label)
	label[LabelKind] = client.GetObjectKind().GroupVersionKind().Kind
	label[LabelNamespace] = client.GetNamespace()
	label[LabelName] = client.GetName()
	label[LabelControllerName] = config.ControllerConfig.ControllerName
	label[LabelManagedBy] = "api7-ingress-controller"
	for i := 0; i < len(args); i += 2 {
		label[args[i]] = args[i+1]
	}
	return label
}
