package featuregates

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TimeTravelFeatureGate = "timeTravel"
	FgConfigMapName       = "osc-feature-gates"
	OperatorNamespace     = "openshift-sandboxed-containers-operator"
)

var DefaultFeatureGates = map[string]bool{
	"timeTravel": false,
}

func IsEnabled(ctx context.Context, client client.Client, feature string) bool {
	cfgMap := &corev1.ConfigMap{}
	err := client.Get(ctx,
		types.NamespacedName{Name: FgConfigMapName, Namespace: OperatorNamespace},
		cfgMap)

	if err != nil {
		log.Printf("Error fetching feature gates: %v", err)
	} else {
		if value, exists := cfgMap.Data[feature]; exists {
			return value == "true"
		}
	}

	defaultValue, exists := DefaultFeatureGates[feature]
	if exists {
		return defaultValue
	}
	return false
}
