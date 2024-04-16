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
	"encoding/json"
	"fmt"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	//"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	apiv1alpha1 "tony123.tw/api/v1alpha1"
	"tony123.tw/handlers"
	handler "tony123.tw/handlers"
	"tony123.tw/util"
)

// MigrationReconciler reconciles a Migration object
type MigrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type kubeletCheckpointResponse struct {
	Items []string `json:"items"`
}

//+kubebuilder:rbac:groups=api.my.domain,resources=migrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Migration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

func (r *MigrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO(user): your logic here

	l := log.FromContext(ctx)

	// get the migration object
	migration := &apiv1alpha1.Migration{}
	err := r.Get(ctx, req.NamespacedName, migration)

	if err != nil {
		l.Error(err, "unable to fetch the migration object")
		return ctrl.Result{}, err
	}
	// check if the deployment field is empty
	if migration.Spec.Deployment == "" {
		CheckpointSinglePod(ctx, r, migration, nil)
	} else {
		CheckpointDeployment(ctx, r, migration)
	}
	// TODO(user): your logic here
	// make install before running the operator, because for api perspective, they don't understand the CRD
	// loop over the pods in default namespace and find the pods with the postgresql image

	return ctrl.Result{}, nil
}
func CheckpointDeployment(ctx context.Context, r *MigrationReconciler, migration *apiv1alpha1.Migration) {
	// check all the pods in the deployment and checkpoint them
	// get the deployment
	deployment := &appsv1.Deployment{}
	// create a namespace that equals to the namespace of the migration object

	namespace := migration.Spec.Namespace
	ns := types.NamespacedName{
		Name:      migration.Spec.Deployment,
		Namespace: namespace,
	}
	err := r.Get(ctx, ns, deployment)
	if err != nil {
		log.Log.Error(err, "unable to get the deployment")
		return
	}

	labels := deployment.Spec.Selector
	labelSelector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		log.Log.Error(err, "unable to get the label selector")
		return
	}

	listOptions := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	// get the pods in the deployment
	CheckpointSinglePod(ctx, r, migration, listOptions)

}

func CheckpointSinglePod(ctx context.Context, r *MigrationReconciler, migration *apiv1alpha1.Migration, listOptions *client.ListOptions) {
	podList := &corev1.PodList{}
	logger := log.FromContext(ctx)
	if listOptions == nil {
		err := r.List(ctx, podList)
		if err != nil {
			logger.Error(err, "unable to list the pods")
			return
		}
	} else {
		err := r.List(ctx, podList, listOptions)
		if err != nil {
			logger.Error(err, "unable to list the pods")
			return
		}
	}

	for _, pod := range podList.Items {
		// checkpoint the container in each pod
		if pod.Status.Phase == corev1.PodRunning {
			for _, container := range pod.Spec.Containers {
				address := fmt.Sprintf(
					"https://%s:%d/checkpoint/%s/%s/%s",
					pod.Status.HostIP,
					10250,
					migration.Spec.Namespace,
					pod.Name,
					container.Name,
				)
				log.Log.Info("checkpoint kubelet", "address", address)
				client := handler.GetKubeletClient()

				resp, err := handler.CheckpointPod(client, address)
				if err != nil {
					log.Log.Error(err, "unable to checkpoint the pod")
					return
				}

				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Log.Info("error while reading body: ", "error", err)
					continue
				}
				kubeletResponse := &kubeletCheckpointResponse{}
				err = json.Unmarshal(body, kubeletResponse)
				// use that checkpoint to build the image
				if err != nil {
					log.Log.Error(err, "Error unmarshalling kubelet response")
					continue
				}
				log.Log.Info("got response", "response", kubeletResponse, "body", string(body))

				clientset,err := util.CreateClientSet()

				if err != nil {
					log.Log.Error(err, "unable to create clientset")
					return
				}
					
				// find the pod ip of registry pod
				registryIp,err := handlers.ReturnRegistryIP(clientset, pod.Spec.NodeName)
				if err != nil {
					log.Log.Error(err, "unable to get the registry ip")
					return
				}

				err = handlers.BuildahPodPushImage(pod.Spec.NodeName, "docker-registry", kubeletResponse.Items[0], registryIp)

				if err != nil {
					log.Log.Error(err, "unable to push image to registry")
					return
				}
			}
		}
	}
}



// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Migration{}).
		Complete(r)
}
