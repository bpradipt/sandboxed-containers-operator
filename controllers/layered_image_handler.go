package controllers

import (
	"context"
	"fmt"

	yaml "github.com/ghodss/yaml"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *KataConfigOpenShiftReconciler) handleLayeredImageDeploymentFeature(state FeatureGateState) error {

	// Check if MachineConfig exists and return the same without changing anything
	mc, err := r.getExistingMachineConfig()
	if err != nil {
		r.Log.Info("Error in getting existing MachineConfig", "err", err)
		return err
	}

	if mc != nil {
		r.Log.Info("MachineConfig is already present. No changes will be done")
		r.ExistingMc = mc
		return nil
	}

	if state == Enabled {
		r.Log.Info("LayeredImageDeployment feature is enabled")

		cm := &corev1.ConfigMap{}
		err := r.Client.Get(context.Background(), types.NamespacedName{
			Name:      "sandboxed-containers-image-cm",
			Namespace: "openshift-sandboxed-containers-operator",
		}, cm)
		if err != nil {
			r.Log.Info("Error in retrieving MC ConfigMap", "err", err)
			return err
		}

		mcYaml, exists := cm.Data["mc.yaml"]
		if !exists {
			return fmt.Errorf("mc.yaml not found in ConfigMap")
		}

		mc = &mcfgv1.MachineConfig{}
		err = yaml.Unmarshal([]byte(mcYaml), mc)
		if err != nil {
			r.Log.Info("Error in unmarshalling MachineConfig yaml", "err", err)
			return err
		}

		r.ImgMc = mc
	} else {
		r.Log.Info("LayeredImageDeployment feature is disabled. Resetting ImgMc")
		// Reset ImgMc
		r.ImgMc = nil
	}

	return nil
}

func (r *KataConfigOpenShiftReconciler) getExistingMachineConfig() (*mcfgv1.MachineConfig, error) {
	r.Log.Info("Getting any existing MachineConfigs related to KataConfig")

	// Retrieve the existing MachineConfig
	// Check for label "app":r.kataConfig.Name
	// and name "50-enable-sandboxed-containers-extension" or name "50-enable-sandboxed-containers-image"
	mcList := &mcfgv1.MachineConfigList{}
	err := r.Client.List(context.Background(), mcList)
	if err != nil {
		r.Log.Info("Error in listing MachineConfigs", "err", err)
		return nil, err
	}

	for _, mc := range mcList.Items {
		if mc.Labels["app"] == r.kataConfig.Name &&
			(mc.Name == extension_mc_name || mc.Name == image_mc_name) {
			return &mc, nil
		}
	}

	r.Log.Info("No existing MachineConfigs related to KataConfig found")

	return nil, nil
}
