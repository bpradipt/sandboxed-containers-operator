/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"os"

	peerpodcontrollers "github.com/confidential-containers/cloud-api-adaptor/src/peerpod-ctrl/controllers"
	peerpodconfigcontrollers "github.com/confidential-containers/cloud-api-adaptor/src/peerpodconfig-ctrl/controllers"
	configv1 "github.com/openshift/api/config/v1"
	secv1 "github.com/openshift/api/security/v1"
	mcfgapi "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	nodeapi "k8s.io/api/node/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// These imports are unused but required in go.mod
	// for caching during manifest generation by controller-gen
	_ "github.com/spf13/cobra"
	_ "sigs.k8s.io/controller-tools/pkg/crd"
	_ "sigs.k8s.io/controller-tools/pkg/genall"
	_ "sigs.k8s.io/controller-tools/pkg/genall/help/pretty"
	_ "sigs.k8s.io/controller-tools/pkg/loader"

	peerpod "github.com/confidential-containers/cloud-api-adaptor/src/peerpod-ctrl/api/v1alpha1"
	peerpodconfig "github.com/confidential-containers/cloud-api-adaptor/src/peerpodconfig-ctrl/api/v1alpha1"
	ccov1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	operatorsapiv2 "github.com/operator-framework/api/pkg/operators/v2"

	kataconfigurationv1 "github.com/openshift/sandboxed-containers-operator/api/v1"
	"github.com/openshift/sandboxed-containers-operator/controllers"
	// +kubebuilder:scaffold:imports
)

const (
	OperatorNamespace = "openshift-sandboxed-containers-operator"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// Struct to hold config status for KataConfig
type ConfigStatus struct {
	Exists         bool
	EnablePeerPods bool
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nodeapi.AddToScheme(scheme))

	utilruntime.Must(secv1.AddToScheme(scheme))

	utilruntime.Must(mcfgapi.Install(scheme))

	utilruntime.Must(kataconfigurationv1.AddToScheme(scheme))

	utilruntime.Must(peerpodconfig.AddToScheme(scheme))

	utilruntime.Must(peerpod.AddToScheme(scheme))

	utilruntime.Must(configv1.AddToScheme(scheme))

	utilruntime.Must(ccov1.AddToScheme(scheme))

	utilruntime.Must(operatorsapiv2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func SetTimeEncoderToRfc3339() zap.Opts {
	return func(o *zap.Options) {
		o.TimeEncoder = zapcore.RFC3339TimeEncoder
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true), SetTimeEncoderToRfc3339()))

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}

	// apiclient.New() returns a client without cache.
	// cache is not initialized before mgr.Start()
	// we need this because we need to interact with OperatorCondition
	apiClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to create apiclient")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: metricsAddr},
		WebhookServer: &webhook.DefaultServer{
			Options: webhook.Options{
				Port: 9443,
			},
		},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "290f4947.kataconfiguration.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	isOpenshift, err := controllers.IsOpenShift()
	if err != nil {
		setupLog.Error(err, "unable to use discovery client")
		os.Exit(1)
	}

	if isOpenshift {

		err = fixScc(context.TODO(), mgr)
		if err != nil {
			setupLog.Error(err, "unable to create SCC")
			os.Exit(1)
		}

		err = labelNamespace(context.TODO(), mgr)
		if err != nil {
			setupLog.Error(err, "unable to add labels to namespace")
			os.Exit(1)
		}

		setupLog.Info("added labels")

		/*
			// Get KataConfig status
			configStatus, err := getKataConfig(context.TODO(), mgr)
			if err != nil {
				setupLog.Error(err, "unable to get KataConfig")
				os.Exit(1)
			}

			setupLog.Info("KataConfig status", "Exists", configStatus.Exists, "EnablePeerPods", configStatus.EnablePeerPods)
		*/

		setupLog.Info("Setting OperatorCondition.")

		upgradeableCondition, err := controllers.NewOperatorCondition(apiClient, operatorsapiv2.Upgradeable)
		if err != nil {
			setupLog.Error(err, "Cannot create the Upgradeable Operator Condition")
			os.Exit(1)
		}

		err = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := upgradeableCondition.Set(context.TODO(), metav1.ConditionFalse, controllers.UpgradeableDisableReason,
				controllers.UpgradeableDisableMessage)
			if err != nil {
				setupLog.Error(err, "Cannot set the status of the Upgradeable Operator Condition")
			}
			return err == nil, nil
		})
		if err != nil {
			setupLog.Error(err, "Cannot set the status of the Upgradeable Operator Condition")
			os.Exit(1)
		}

		/*
			// re-create the condition, this time with the final client
			upgradeableCondition, err = controllers.NewOperatorCondition(mgr.GetClient(), operatorsapiv2.Upgradeable)
			if err != nil {
				setupLog.Error(err, "Cannot create the Upgradeable Operator Condition")
				os.Exit(1)
			}
		*/

		if err = (&controllers.KataConfigOpenShiftReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KataConfig"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create KataConfig controller for OpenShift cluster", "controller", "KataConfig")
			os.Exit(1)
		}

		if err = (&peerpodconfigcontrollers.PeerPodConfigReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("RemotePodConfig"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create RemotePodConfig controller for OpenShift cluster", "controller", "RemotePodConfig")
			os.Exit(1)
		}

		if err = (&peerpodcontrollers.PeerPodReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			// setting nil will delegate Provider creation to reconcile time, make sure RBAC permits:
			//+kubebuilder:rbac:groups="",resourceNames=peer-pods-cm;peer-pods-secret,resources=configmaps;secrets,verbs=get
			Provider: nil,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create peerpod resources controller", "controller", "PeerPod")
			os.Exit(1)
		}

	}

	if err = (&kataconfigurationv1.KataConfig{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "KataConfig")
		os.Exit(1)
	}

	if err = (&controllers.SecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Credentials"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Credentials")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func fixScc(ctx context.Context, mgr manager.Manager) error {

	scc := controllers.GetScc()
	err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(scc), scc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Nothing to do.
			err = nil
		}
	} else if scc.SELinuxContext.Type == secv1.SELinuxStrategyMustRunAs {
		// A 1.2-style SCC breaks the MCO. This was fixed by
		// commit d4745883e38f, i.e. OSC >= 1.3 doesn't create
		// broken SCC anymore, but an existing instance still
		// needs to be fixed.
		setupLog.Info("Fixing SCC")
		scc.SELinuxContext = secv1.SELinuxContextStrategyOptions{
			Type: secv1.SELinuxStrategyRunAsAny,
		}
		err = mgr.GetClient().Update(ctx, scc)
	}

	return err
}

func labelNamespace(ctx context.Context, mgr manager.Manager) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorNamespace,
		},
	}
	err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(ns), ns)
	if err != nil {
		setupLog.Error(err, "Unable to add label to the namespace")
		return err
	}

	setupLog.Info("Labelling Namespace")
	setupLog.Info("Labels: ", "Labels", ns.ObjectMeta.Labels)
	// Add namespace label to align with newly introduced Pod Security Admission controller
	ns.ObjectMeta.Labels["openshift.io/cluster-monitoring"] = "true"
	ns.ObjectMeta.Labels["pod-security.kubernetes.io/enforce"] = "privileged"
	ns.ObjectMeta.Labels["pod-security.kubernetes.io/audit"] = "privileged"
	ns.ObjectMeta.Labels["pod-security.kubernetes.io/warn"] = "privileged"

	return mgr.GetClient().Update(ctx, ns)
}

// Retrieve KataConfig and update ConfigStatus
func getKataConfig(ctx context.Context, mgr manager.Manager) (ConfigStatus, error) {
	configStatus := ConfigStatus{}

	kataConfig := &kataconfigurationv1.KataConfig{}
	err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(kataConfig), kataConfig)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// KataConfig CRD does not exist
			configStatus.Exists = false
			setupLog.Info("KataConfig CRD does not exist")
			return configStatus, nil
		}
		return configStatus, err
	}

	configStatus.Exists = true
	configStatus.EnablePeerPods = kataConfig.Spec.EnablePeerPods

	setupLog.Info("ConfigStatus", "KataConfig", configStatus.Exists, "EnablePeerPods", configStatus.EnablePeerPods)
	return configStatus, nil
}

// Retrieve OpenShift version
func getOpenShiftVersion(ctx context.Context, mgr manager.Manager) (string, error) {

	setupLog.Info("GetOpenShiftVersion")
	return "", nil
}

// Create OperatorStatus condition
