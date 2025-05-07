package cache

import (
	"fmt"
	"strings"

	"github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/internal/controller/label"
)

const (
	KindLabelIndex = "label"
)

/*
var KindLabelIndexer = LabelIndexer{
	LabelKeys: []string{label.LabelKind, label.LabelName, label.LabelNamespace},
	GetLabels: func(obj adc.Object) map[string]string {
		return obj.GetLabels()
	},
}
*/

var (
	KindLabelIndexer = LabelIndexer{
		LabelKeys: []string{label.LabelKind, label.LabelNamespace, label.LabelName},
		GetLabels: func(obj any) map[string]string {
			o, ok := obj.(adc.Object)
			if !ok {
				return nil
			}
			return o.GetLabels()
		},
	}
)

type LabelIndexer struct {
	LabelKeys []string
	GetLabels func(obj any) map[string]string
}

func (emi *LabelIndexer) FromObject(obj any) (bool, []byte, error) {
	labels := emi.GetLabels(obj)
	var labelValues []string
	for _, key := range emi.LabelKeys {
		if value, exists := labels[key]; exists {
			labelValues = append(labelValues, value)
		}
	}

	if len(labelValues) == 0 {
		return false, nil, nil
	}

	return true, []byte(strings.Join(labelValues, "/")), nil
}

func (emi *LabelIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != len(emi.LabelKeys) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(emi.LabelKeys), len(args))
	}

	labelValues := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("argument is not a string")
		}
		labelValues = append(labelValues, value)
	}

	return []byte(strings.Join(labelValues, "/")), nil
}
