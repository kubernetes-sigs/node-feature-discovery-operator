/*
Copyright 2021. The Kubernetes Authors.

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
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nfdkubernetesiov1 "github.com/kubernetes-sigs/node-feature-discovery-operator/api/v1"
	"github.com/kubernetes-sigs/node-feature-discovery-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	// scheme holds a new scheme for the operator
	scheme = runtime.NewScheme()

	// setupLog will be used for logging the operator "setup" process so that users know
	// what parts of the logging are associated with the setup of the manager and
	// controller
	setupLog = ctrl.Log.WithName("setup")
)

// init sets up the Go client and NFD schemes. The function "utilruntime.Must" is used to
// panic on non-nil errors that could occur when adding the scheme, as opposed to just letting
// an error occur without properly handling it.
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nfdkubernetesiov1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	// metricsAddr is used by Prometheus to gather the NFD Operator resource usage data.
	// The bind address tells Prometheus which port to scrape this data's metrics from.
	var metricsAddr string

	// enableLeaderElection should be set to 'disable' by default If we enable leader
	// election, then only one node can run the controller manager and we will not
	// have NFD Operator running on all nodes.
	var enableLeaderElection bool

	// probeAddr is responsible for the health probe bind address, where the health
	// probe is responsible for determining liveness, readiness, and configuration
	// of the operator pods.
	var probeAddr string

	// The following 3 lines setup the CLI arguments that are used upon initilization of
	// the operator
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// opts is created using zap to set the operator's logging to "development mode".
	// This mode makes DPanic-level logs panic instead of just logging error events as
	// errors. The settings are then bound to the CLI flag args and the flag args are
	// then parsed.
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Create a new manager to manage the operator and bind its address to port 9443.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "39f5e5c3.nodefeaturediscoveries.nfd.kubernetes.io",
	})

	// If the manager could not be started, then log the error as an operator setup error
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Try to create an NFD reconciler using the manager we just created. Note that we do
	// create a different logger for the NFD reconciler with the name "NodeFeatureDiscovery"
	// and "controllers" in the name to let users know which step in the operator setup is
	// occurring. If this step succeeds, then everything related to controllers will be logged
	// in reference to  "controllers." If this step fails, everything will be logged to the
	// "setup" logger defined at the top of this file.
	if err = (&controllers.NodeFeatureDiscoveryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NodeFeatureDiscovery"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NodeFeatureDiscovery")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// Next, add a Healthz checker to the manager. Healthz is a health and liveness package
	// that the operator will use to periodically check the health of its pods, etc.
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	// Now add a ReadyZ checker to the manager as well. It is important to ensure that the
	// API server's readiness is checked when the operator is installed and running.
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Now that the manager and controller are all setup, it's time to start the manager.
	// TheSetupSignalHandler registers for SIGINT and SIGTERM, which can be used to
	// terminate the manager if they are both called.
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
