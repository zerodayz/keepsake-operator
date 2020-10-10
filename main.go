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
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"        // IMPORTANT: controller-runtime
	"sigs.k8s.io/controller-runtime/pkg/log/zap" // IMPORTANT: controller-runtime logging

	// This imports will be populated by kubebuilder.
	keepsakev1alpha1 "github.com/zerodayz/keepsake-operator/api/v1alpha1"
	"github.com/zerodayz/keepsake-operator/controllers"
	// +kubebuilder:scaffold:imports
)

// Every set of controllers needs a Scheme, which provides mappings between Kinds and their corresponding Go types.
// We’ll talk a bit more about Kinds when we write our API definition, so just keep this in mind for later.
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// This scheme will be populated by kubebuilder.
	utilruntime.Must(keepsakev1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,

		//	Watching resources in all Namespaces (default)
		//
		//	A Manager is initialized with no Namespace option specified, or Namespace: "" will watch all Namespaces:
		//
		//	...
		//	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		//	Scheme:             scheme,
		//	MetricsBindAddress: metricsAddr,
		//	Port:               9443,
		//	LeaderElection:     enableLeaderElection,
		//	LeaderElectionID:   "f1c5ece8.example.com",
		//})
		//	...

		//	Watching resources in a single Namespace
		//
		//	To restrict the scope of the Manager’s cache to a specific Namespace set the Namespace field in Options:
		//
		//	...
		//	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		//	Scheme:             scheme,
		//	Namespace:          "operator-namespace",
		//	MetricsBindAddress: metricsAddr,
		//})
		//	...

		//	Watching resources in a set of Namespaces
		//
		//	It is possible to use MultiNamespacedCacheBuilder from Options to watch and manage resources in a set of Namespaces:
		//
		//	...
		//	namespaces := []string{"foo", "bar"} // List of Namespaces
		//
		//	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		//	Scheme:             scheme,
		//	NewCache:           cache.MultiNamespacedCacheBuilder(namespaces),
		//	MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		//})
		//	...

		//	In the above example, a CR created in a Namespace not in the set passed to Options will not be reconciled
		//	by its controller because the Manager does not manage that Namespace.
		//
		//	IMPORTANT: Note that this is not intended to be used for excluding Namespaces,
		//	this is better done via a Predicate.

		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "f9b63cfb.example.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// This builder will be populated by kubebuilder.
	if err = (&controllers.KeepsakeReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Keepsake"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Keepsake")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
