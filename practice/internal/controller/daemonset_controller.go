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
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=daemonsets/finalizers,verbs=update

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
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	// detect the docker registry daemonset has received the new image at the timestamp

	// get the pods from namespace docker-registry
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace("docker-registry")); err != nil {
		log.Error(err, "unable to list pods")
		return ctrl.Result{}, err
	}

	// get the pods whose prefix name is docker-registry
	for _, pod := range pods.Items {
		log.Info("Pod name", "name", pod.Name)
		// if pod.Name includes docker-registry
		if strings.Contains(pod.Name, "docker-registry") {
			// get the pod ip
			podIP := pod.Status.PodIP
			// list all the repositories in the registry
			// curl -X GET <user>:<pass> https://podIP:5000/v2/_catalog
			// user: myuser password: mypassword
			
			// create the request
			req, err := http.NewRequest("GET", "https://"+podIP+":5000/v2/_catalog", nil)
			// with user and password
			req.SetBasicAuth("myuser",
				"mypassword")

			// send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Error(err, "unable to send the request")
				return ctrl.Result{}, err
			}
			defer resp.Body.Close()
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.

func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// watch daemonset for docker-registry
		For(&appsv1.DaemonSet{}).
		Complete(r)
}
