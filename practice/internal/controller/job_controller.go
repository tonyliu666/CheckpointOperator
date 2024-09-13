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

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"tony123.tw/handlers"
	"tony123.tw/restore"
	util "tony123.tw/util"

	// "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	finalizerName = "checkpoint-image/cleanup"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	var job batchv1.Job

	// Fetch the Job instance in migration namespace
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the Job is marked for deletion
	if job.DeletionTimestamp != nil && containsString(job.Finalizers, finalizerName) {
		// Notify other pods before deletion
		log.Log.Info("Notifying pods before deletion")

		// Remove the finalizer to allow deletion to proceed
		job.Finalizers = removeString(job.Finalizers, finalizerName)
		err := r.Update(ctx, &job)
		if err != nil {
			log.Log.Error(err, "unable to update the job")
			return ctrl.Result{}, err
		}

		// before deletion, deploy the new pod on the destination node
		podName, err := handlers.DeployPodOnNewNode()
		if err != nil {
			if err.Error() == "no more pods to deploy" {
				log.Log.Info("No more pods to deploy")
				return ctrl.Result{}, nil
			}
			log.Log.Error(err, "Error deploying pod on new node")
			return ctrl.Result{}, err
		}

		// delete the old pod
		err = restore.DeleteOldPod(util.SourceNamespace, podName)
		if err != nil {
			log.Log.Error(err, "unable to delete the old pod")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Complete(r)
}



func containsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func removeString (slice []string, str string) []string {
	for i, v := range slice {
		if v == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
