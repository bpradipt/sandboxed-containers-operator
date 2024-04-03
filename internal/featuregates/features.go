package featuregates

/* Design aspects of implementing feature gates

- feature gate is only for experimental features
- Keep the feature gate as simple boolean.
- Any config specific for a feature gate should be in it’s own configMap. This aligns with our current implementation of peer-pods and image generator feature
- When we decide to make the feature as a stable feature, it should move to kataConfig.spec
*/

const (
	FeatureGatesConfigMapName     = "osc-feature-gates"
	ImageBasedDeployment          = "imageBasedDeployment"
	ImageBasedDeploymentConfigMap = "osc-feature-gate-image-deploy-config"
)

// Sample ConfigMap with Features

/*
apiVersion: v1
kind: ConfigMap
metadata:
  name: osc-feature-gates
  namespace: openshift-sandboxed-containers-operator
data:
  imageBasedDeployment: "false"
  additionalRuntimeClasses: "false"

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: osc-feature-gate-image-deploy-config
  namespace: openshift-sandboxed-containers-operator
data:
  osImageURL="quay.io/...."
  kernelArguments="a=b c=d ..."
*/

// Get the feature gate configmap name from the feature gate name
func GetFeatureGateConfigMapName(feature string) string {
	switch feature {
	case ImageBasedDeployment:
		return ImageBasedDeploymentConfigMap
	default:
		return ""
	}
}

// Method to check if the configmap name is one of
// FeatureGatesConfigMapName, ImageBasedDeploymentConfigMap
func IsFeatureGateConfigMap(name string) bool {
	switch name {
	case FeatureGatesConfigMapName, ImageBasedDeploymentConfigMap:
		return true
	default:
		return false
	}
}
