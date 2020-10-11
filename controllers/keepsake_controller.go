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

package controllers

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keepsakev1alpha1 "github.com/zerodayz/keepsake-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeepsakeReconciler reconciles a Keepsake object
type KeepsakeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=keepsake.example.com,resources=keepsakes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keepsake.example.com,resources=keepsakes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *KeepsakeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("keepsake", req.NamespacedName)

	// Fetch the Keepsake instance
	keepsake := &keepsakev1alpha1.Keepsake{}
	err := r.Get(ctx, req.NamespacedName, keepsake)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Keepsake resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Keepsake")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: keepsake.Name, Namespace: keepsake.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForKeepsake(keepsake)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := keepsake.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the Keepsake status with the pod names
	// List the pods for this keepsake's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(keepsake.Namespace),
		client.MatchingLabels(labelsForKeepsake(keepsake.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Keepsake.Namespace", keepsake.Namespace, "Keepsake.Name", keepsake.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, keepsake.Status.Nodes) {
		keepsake.Status.Nodes = podNames
		err := r.Status().Update(ctx, keepsake)
		if err != nil {
			log.Error(err, "Failed to update Keepsake status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *KeepsakeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keepsakev1alpha1.Keepsake{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// deploymentForKeepsake returns a keepsake Deployment object
func (r *KeepsakeReconciler) deploymentForKeepsake(m *keepsakev1alpha1.Keepsake) *appsv1.Deployment {
	ls := labelsForKeepsake(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "quay.io/zerodayz/keepsake:latest",
						Name:    "keepsake",
						Command: []string{"wiki"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "keepsake",
						}},
						Env: []corev1.EnvVar{{
							Name:	"KEEPSAKE_DISABLE_SSL",
							Value:	"1",
						}},
					},
					{
						Image:   "mariadb:latest",
						Name:    "keepsake-mysql",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3306,
							Name:          "keepsake-mysql",
						}},
						Env: []corev1.EnvVar{{
							Name:	"MYSQL_ROOT_PASSWORD",
							Value:	"roottoor",
						},
						{
							Name:	"MYSQL_DATABASE",
							Value:	"gowiki",
						},
						{
							Name:	"MYSQL_USER",
							Value:	"gowiki",
						},
						{
							Name:	"MYSQL_PASSWORD",
							Value:	"gowiki55",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:	"keepsake-mysql-data",
							MountPath: "/var/lib/mysql",
						}},
					}},
				},
			},
		},
	}
	// Set Keepsake instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForKeepsake returns the labels for selecting the resources
// belonging to the given keepsake CR name.
func labelsForKeepsake(name string) map[string]string {
	return map[string]string{"app": "keepsake", "keepsake_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
