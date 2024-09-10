/*
Copyright 2024.

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

package controller

import (
	"context"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"tony123.tw/handlers"
)

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1,resources=namespaces,verbs=create;get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DaemonSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	// get the pods from namespace docker-registry
	pods := &corev1.PodList{}

	// get the pods whose label is app: docker-registry
	if err := r.List(ctx, pods, client.InNamespace("docker-registry"), client.MatchingLabels{"app": "docker-registry"}); err != nil {
		log.Log.Error(err, "unable to list pods")
		return ctrl.Result{RequeueAfter: 500 * time.Second}, err
	}
	// depend on how many docker registry pods, create how many go routines
	var wg sync.WaitGroup
	wg.Add(len(pods.Items))
	for _, pod := range pods.Items {
		go func(pod corev1.Pod) {
			defer wg.Done() // Ensure Done is called
			if err := handlers.DeployPodOnNewNode(&pod); err != nil {
				log.Log.Error(err, "unable to deploy pod")
			}
		}(pod) // Pass pod as argument to avoid closure capturing issue
	}

	wg.Wait()
	// return ctrl.Result{}, nil
	return ctrl.Result{RequeueAfter: 500 * time.Millisecond}, nil
}

// SetupWithManager sets up the controller with the Manager.

func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// watch daemonset for docker-registry
		For(&appsv1.DaemonSet{
			// label selector select the daemonset whose label is app: docker-registry
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "docker-registry"},
				},
			},
		}).
		Complete(r)
}
