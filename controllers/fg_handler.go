package controllers

import (
	"context"

	"github.com/openshift/sandboxed-containers-operator/internal/featuregates"
)

// Create two enums to represent the state of the feature gates
type FeatureGateState int

const (
	Enabled FeatureGateState = iota
	Disabled
)

// Function to handle the feature gates
func (r *KataConfigOpenShiftReconciler) processFeatureGates() error {

	// Check which feature gates are enabled in the FG ConfigMap and
	// perform the necessary actions
	// The feature gates are defined in internal/featuregates/featuregates.go
	// and are fetched from the ConfigMap in the namespace
	// Eg. TimeTravelFeatureGate

	if featuregates.IsEnabled(context.TODO(), featuregates.TimeTravelFeatureGate) {
		r.Log.Info("Feature gate is enabled", "featuregate", featuregates.TimeTravelFeatureGate)
		// Perform the necessary actions
		r.handleTimeTravelFeature(Enabled)
	} else {
		r.Log.Info("Feature gate is disabled", "featuregate", featuregates.TimeTravelFeatureGate)
		// Perform the necessary actions
		r.handleTimeTravelFeature(Disabled)
	}

	return nil

}

// Function to handle the TimeTravel feature gate
func (r *KataConfigOpenShiftReconciler) handleTimeTravelFeature(state FeatureGateState) {
	// Perform the necessary actions for the TimeTravel feature gate
	if state == Enabled {
		r.Log.Info("Starting TimeTravel")
	} else {
		r.Log.Info("Stopping TimeTravel")
	}
}
