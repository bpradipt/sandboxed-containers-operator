package featuregates

import (
	"context"

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

type FeatureGateStatus struct {
	FeatureGates map[string]bool
}

// This method returns a new FeatureGateStatus object
// that contains the status of the feature gates
// defined in the ConfigMap in the namespace
// If there are any errors in retrieving the ConfigMap, or the feature gates
// are not defined in the ConfigMap, the default values are used
func NewFeatureGateStatus(client client.Client) (*FeatureGateStatus, error) {
	fgStatus := &FeatureGateStatus{
		FeatureGates: make(map[string]bool),
	}

	cfgMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: FgConfigMapName,
		Namespace: OperatorNamespace}, cfgMap)
	if err == nil {
		for feature, value := range cfgMap.Data {
			fgStatus.FeatureGates[feature] = value == "true"
		}
	}

	// Add default values for missing feature gates
	for feature, defaultValue := range DefaultFeatureGates {
		if _, exists := fgStatus.FeatureGates[feature]; !exists {
			fgStatus.FeatureGates[feature] = defaultValue
		}
	}
	return fgStatus, err
}

func IsEnabled(fgStatus *FeatureGateStatus, feature string) bool {

	return fgStatus.FeatureGates[feature]
}
