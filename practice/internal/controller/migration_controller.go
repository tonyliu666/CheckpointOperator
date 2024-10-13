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
	"strings"
	"time"

	"github.com/go-logr/logr"
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
	"tony123.tw/util"
)

// MigrationReconciler reconciles a Migration object
type MigrationReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	migrationSpec *apiv1alpha1.MigrationSpec
}

type kubeletCheckpointResponse struct {
	Items []string `json:"items"`
}

//+kubebuilder:rbac:groups=api.my.domain,resources=migrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;patch;delete

// want the controller to list all the pods in other namespace and checkpoint them

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
	r.migrationSpec = &migration.Spec

	// fill the variable in util/global.go
	util.FillinGlobalVariables(r.migrationSpec.PodName, r.migrationSpec.Deployment, r.migrationSpec.Namespace, r.migrationSpec.DestinationNode, r.migrationSpec.DestinationNamespace, r.migrationSpec.Specify)

	l.Info("pod needs to be checkpointed", "podName", r.migrationSpec.PodName)
	l.Info("ProcessPodsMap", "podName", util.ProcessPodsMap)
	if err != nil {
		l.Error(err, "unable to fetch the migration object")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	if len(r.migrationSpec.Specify) != 0 {
		// checkpoint the specified pods in the given namespace
		listOptions := &client.ListOptions{
			Namespace: r.migrationSpec.Namespace,
		}
		err = r.checkpointSinglePod(ctx, listOptions, true)
		if err != nil {
			l.Error(err, "unable to checkpoint the specified pods")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	} else {
		// check if the deployment field is empty
		if r.migrationSpec.Deployment != "" {
			r.checkpointDeployment(ctx)
		} else {
			r.checkpointSinglePod(ctx, nil, false)
		}
	}
	// TODO(user): your logic here
	// make install before running the operator, because for api perspective, they don't understand the CRD
	// loop over the pods in default namespace and find the pods with the postgresql image

	return ctrl.Result{}, nil
}
func (r *MigrationReconciler) checkpointDeployment(ctx context.Context) error {
	// check all the pods in the deployment and checkpoint them
	// get the deployment
	deployment := &appsv1.Deployment{}
	// create a namespace that equals to the namespace of the migration object

	namespace := r.migrationSpec.Namespace
	ns := types.NamespacedName{
		Name:      r.migrationSpec.Deployment,
		Namespace: namespace,
	}
	err := r.Get(ctx, ns, deployment)
	if err != nil {
		log.Log.Error(err, "unable to get the deployment")
		return err
	}

	labels := deployment.Spec.Selector
	labelSelector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		log.Log.Error(err, "unable to get the label selector")
		return err
	}

	listOptions := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}
	// get the pods in the deployment
	err = r.checkpointSinglePod(ctx, listOptions, false)
	if err != nil {
		log.Log.Error(err, "unable to checkpoint the deployment")
		return err
	}
	return nil

}

func (r *MigrationReconciler) checkpointSinglePod(ctx context.Context, listOptions *client.ListOptions, specifyOrNot bool) error {
	podList := &corev1.PodList{}
	logger := log.FromContext(ctx)
	err := r.filterPods(listOptions, podList, ctx, logger, specifyOrNot)
	if err != nil {
		logger.Error(err, "unable to filter the pods")
		return err
	}

	for _, pod := range podList.Items {
		// checkpoint the container in each pod
		if pod.Status.Phase == corev1.PodRunning {
			for _, container := range pod.Spec.Containers {
				address := fmt.Sprintf(
					"https://%s:%d/checkpoint/%s/%s/%s",
					pod.Status.HostIP,
					10250,
					pod.Namespace,
					pod.Name,
					container.Name,
				)
				log.Log.Info("checkpoint kubelet", "address", address)
				client := handlers.GetKubeletClient()
				resp, err := handlers.CheckpointPod(client, address)
				if err != nil {
					log.Log.Error(err, "unable to checkpoint the pod")
					return err
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

				clientset, err := util.CreateClientSet()

				if err != nil {
					log.Log.Error(err, "unable to create clientset")
					return err
				}

				// find the pod ip of registry pod
				registryIp, err := handlers.ReturnRegistryIP(clientset, r.migrationSpec.DestinationNamespace)
				if err != nil {
					log.Log.Error(err, "unable to get the registry ip")
					return err
				}
				// destination node should pull the original image(eg: postgresql:latest)
				err = handlers.OriginalImageChecker(&pod, r.migrationSpec.DestinationNode)
				if err != nil {
					log.Log.Error(err, "unable to pull the original image")
					return err
				}

				// buildah deployment deployed on the node which is same as the node of the pod
				err = handlers.BuildahPodPushImage(pod.Spec.NodeName, "docker-registry", kubeletResponse.Items[0], registryIp)
				if err != nil {
					log.Log.Error(err, "unable to push image to registry")
					return err
				}

				err = handlers.DeleteBuildahJobs(clientset)
				if err != nil {
					log.Log.Error(err, "unable to delete the buildah jobs")
					return err
				}
			}
		}
	}
	return nil
}

func (r *MigrationReconciler) filterPods(listOptions *client.ListOptions, podList *corev1.PodList,
	ctx context.Context, logger logr.Logger, specifyOrNot bool) error {

	if listOptions == nil {
		err := r.List(ctx, podList)
		if err != nil {
			logger.Error(err, "unable to list the pods")
			return err
		}
		// only keep the pod whose name is the same as the podname in the migration object
		for _, pod := range podList.Items {
			if pod.Name == r.migrationSpec.PodName {
				// only keep the pod whose name is the same as the podname in the migration object
				podList.Items = []corev1.Pod{pod}
			}
		}
	} else {
		if specifyOrNot {
			newpodList := &corev1.PodList{}
			err := r.List(ctx, newpodList, listOptions)
			if err != nil {
				logger.Error(err, "unable to list the pods")
				return err
			}
			for _, pod := range r.migrationSpec.Specify {
				found := false
				for i, podItem := range newpodList.Items {
					// if the podItem.Name contains the pod name, then add it to the newPodList
					if podItem.Name == pod {
						// append the pod to the podList
						podList.Items = append(podList.Items, newpodList.Items[i])
						found = true
					}
				}
				if !found {
					logger.Info("pod not found in your specified pods", "pod", pod)
					return fmt.Errorf("pod not found in your specified pods")
				}
			}
		} else {
			filteredPods := []corev1.Pod{}
			err := r.List(ctx, podList, listOptions)
			if err != nil {
				logger.Error(err, "unable to list the pods")
				return err
			}
			// Now I can't handle this case: podname: nginx, deployment: nginx recorded in custom resource, the nginx-deployment will also be checkpointed
			for i, pod := range podList.Items {
				// TODO: add the pods name + pods namespace as a key in the util.ProcessPodsMap
				// Because the validation webhook doesn't check the pods in the deployment
				util.FillinGlobalVariables(pod.Name, "", r.migrationSpec.Namespace, r.migrationSpec.DestinationNode, r.migrationSpec.DestinationNamespace, r.migrationSpec.Specify)
				if strings.HasPrefix(pod.Name, r.migrationSpec.PodName) {
					filteredPods = append(filteredPods, podList.Items[i])
				}
			}
			podList.Items = filteredPods
		}

	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Migration{}).
		Owns(&apiv1alpha1.Migration{}). // add Owns function here
		Complete(r)
}
