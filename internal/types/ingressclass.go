package types

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressClass struct {
	client.Object
}
