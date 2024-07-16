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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	//"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "tony123.tw/api/v1alpha1"
	"tony123.tw/handlers"
	util "tony123.tw/util"
	config "tony123.tw/util/config"
)

// MigrationReconciler reconciles a Migration object
type MigrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type kubeletCheckpointResponse struct {
	Items []string `json:"items"`
}

// define the global variable of the migration
var Migration *apiv1alpha1.Migration

//+kubebuilder:rbac:groups=api.my.domain,resources=migrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=migrations/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
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

// createPersistentVolume creates a PersistentVolume
func createPersistentVolume(index int, nfsServer, path string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-checkpoints" + fmt.Sprintf("%d", index),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("40Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadOnlyMany,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: nfsServer,
					Path:   path,
				},
			},
		},
	}
}

// createPersistentVolumeClaim creates a PersistentVolumeClaim
func createPersistentVolumeClaim(index int, namespace string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-checkpoints" + fmt.Sprintf("%d", index),
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadOnlyMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("40Gi"),
				},
			},
		},
	}
}

// before Reconcile function starts to work, should check whether each pv,pvc installed on each node.
// if not, install them first
func init() {
	// Initialize the maps
	config.PvSourceMap = make(map[string]*corev1.PersistentVolume)
	config.PvcSourceMap = make(map[string]*corev1.PersistentVolumeClaim)

	// Create the PV if it doesn't exist
	// get the nodes ip address
	nodeIPs, err := util.GetAllNodeIPs()
	if err != nil {
		log.Log.Error(err, "unable to get the node ip addresses")
	}
	clientset, err := util.CreateClientSet()
	if err != nil {
		log.Log.Error(err, "unable to create clientset")
	}
	for i, nodeIP := range nodeIPs {
		nfsServer := nodeIP
		nfsPath := "/var/lib/kubelet/checkpoints"
		// create the pv
		log.Log.Info("Creating PV", "nfsServer", nfsServer, "nfsPath", nfsPath)
		pv := createPersistentVolume(i, nfsServer, nfsPath)
		// create the pvc
		pvc := createPersistentVolumeClaim(i, "migration")
		_, err = clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
		if err != nil {
			log.Log.Error(err, "unable to create the pv")
		}
		_, err = clientset.CoreV1().PersistentVolumeClaims("migration").Create(context.Background(), pvc, metav1.CreateOptions{})
		if err != nil {
			log.Log.Error(err, "unable to create the pvc")
		}
		config.PvSourceMap[nfsServer] = pv
		config.PvcSourceMap[nfsServer] = pvc
	}
}

func (r *MigrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	l := log.FromContext(ctx)
	// get the migration object
	Migration := &apiv1alpha1.Migration{}
	err := r.Get(ctx, req.NamespacedName, Migration)

	if err != nil {
		l.Error(err, "unable to fetch the migration object")
		return ctrl.Result{}, err
	}
	// check if the deployment field is empty
	if Migration.Spec.Deployment == "" {
		CheckpointSinglePod(ctx, r, Migration, nil)
	} else {
		CheckpointDeployment(ctx, r, Migration)
	}
	// TODO(user): your logic here
	// make install before running the operator, because for api perspective, they don't understand the CRD
	// loop over the pods in default namespace and find the pods with the postgresql image

	return ctrl.Result{}, nil
}
func CheckpointDeployment(ctx context.Context, r *MigrationReconciler, migration *apiv1alpha1.Migration) error {
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
		return err
	}

	labels := deployment.Spec.Selector
	labelSelector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		fmt.Println("unable to get the label selector")
		log.Log.Error(err, "unable to get the label selector")
		return err
	}

	listOptions := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}
	// get the pods in the deployment
	err = CheckpointSinglePod(ctx, r, migration, listOptions)
	if err != nil {
		log.Log.Error(err, "unable to checkpoint the deployment")
		return err
	}
	return nil
}

func CheckpointSinglePod(ctx context.Context, r *MigrationReconciler, migration *apiv1alpha1.Migration, listOptions *client.ListOptions) error {
	podList := &corev1.PodList{}
	logger := log.FromContext(ctx)
	if listOptions == nil {
		err := r.List(ctx, podList)
		if err != nil {
			logger.Error(err, "unable to list the pods")
			return err
		}
		// only keep the pod whose name is the same as the podname in the migration object
		for _, pod := range podList.Items {
			if pod.Name == migration.Spec.Podname {
				podList = &corev1.PodList{
					Items: []corev1.Pod{pod},
				}
				break
			}
		}
	} else {
		filteredPods := []corev1.Pod{}
		err := r.List(ctx, podList, listOptions)
		if err != nil {
			fmt.Println("unable to list the pods")
			logger.Error(err, "unable to list the pods")
			return err
		}
		// Now I can't handle this case: podname: nginx, deployment: nginx recorded in custom resource, the nginx-deployment will also be checkpointed
		for i, pod := range podList.Items {
			if strings.HasPrefix(pod.Name, migration.Spec.Deployment) {
				filteredPods = append(filteredPods, podList.Items[i])
			}
		}
		podList = &corev1.PodList{
			Items: filteredPods,
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

				// get the source node ip address
				srcNodeIP, err := util.GetNodeIP(pod.Spec.NodeName)
				if err != nil {
					log.Log.Error(err, "unable to get the node ip address")
					return err
				}

				// create buildah pod to build the image
				err = handlers.BuildahPodPushImage(pod.Name, "migration", kubeletResponse.Items[0], srcNodeIP, migration.Spec.Destination)

				if err != nil {
					log.Log.Error(err, "unable to build the image")
					return err
				}
			}
			// before deploying a new pod on the new node,I should examine whether the newnode has the original image
			err :=handlers.OriginalImageChecker(&pod, migration.Spec.Destination)
			if err != nil {
				log.Log.Error(err, "the original image doesn't exist on the destination node")
				return err
			}
			

			// ready to deploy the pod on the destination node
			err = handlers.DeployPodOnNewNode(&pod, migration.Spec.Namespace, migration.Spec.Destination)
			if err != nil {
				log.Log.Error(err, "unable to deploy the pod on the destination")
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Migration{}).
		Complete(r)
}
