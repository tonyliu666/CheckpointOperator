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
	"os"
	"strings"
	"time"

	// "time"

	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	apiv1alpha1 "tony123.tw/api/v1alpha1"
	handler "tony123.tw/handlers"
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

				resp, err := CheckpointPod(client, address)
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

				clientset := CreateClientSet()

				// Create a new deployment with image buildah
				// docker run -–privileged -v ./image/:/image -ti quay.io/buildah/stable /bin/bash
				// bind mount the checkpointed image to the buildah container

				// find the pod ip of registry pod
				registryIp := ReturnRegistryIP(clientset, pod.Spec.NodeName)
				deployment := CreateBuildahDeployment(pod.Spec.NodeName, kubeletResponse, registryIp)

				createdDeployment, err := clientset.AppsV1().Deployments(migration.Spec.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
				if err != nil {
					panic(err.Error())
				}
				logger.Info("Deployment %q created\n", createdDeployment.GetObjectMeta().GetName())
			}
		}
	}
}

func CheckpointPod(client *http.Client, address string) (*http.Response, error) {
	logger := log.Log
	CheckpointStartTime := time.Now()
	resp, err := client.Post(address, "application/json", strings.NewReader(""))
	CheckpointEndTime := time.Now()
	CheckpointDuration := CheckpointEndTime.Sub(CheckpointStartTime).Milliseconds()
	logger.Info("Checkpoint Duration: ", "Duration", CheckpointDuration)
	// err now is facing the problem that the status code is 401 unauthorized
	if err != nil {
		logger.Error(err, "unable to send the request")
		return nil, err
	}
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		logger.Error(err, "unable to checkpoint the container")
		return nil, fmt.Errorf("unable to checkpoint the container")
	}
	// check the response status code
	if resp.StatusCode != http.StatusOK {
		logger.Error(err, "unable to checkpoint the container")
		return nil, fmt.Errorf("unable to checkpoint the container")
	}
	return resp, nil
}

func CreateClientSet() *kubernetes.Clientset {
	// get the kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If running outside the cluster, use kubeconfig file
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}
func ReturnRegistryIP(clientset *kubernetes.Clientset, nodeName string) string {
	// find the registry pod which is on the same node as the pod
	registryPodList, err := clientset.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=docker-registry",
	})
	if err != nil {
		panic(err.Error())
	}
	// find the registry pod which is on the same node as the pod
	var registryPod corev1.Pod
	for _, registryPod = range registryPodList.Items {
		if registryPod.Spec.NodeName == nodeName {
			break
		}
	}
	return registryPod.Status.PodIP

}
func CreateBuildahDeployment(NodeName string, kubeletResponse *kubeletCheckpointResponse, registryIp string) *appsv1.Deployment {
	num := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildah-deployment",
			// set the same namespace as the pod

		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &num,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "buildah",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "buildah",
					},
				},
				// specify the pod should be created on the node where the pod is running
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							// privileged container
							Name:  "buildah",
							Image: "quay.io/buildah/stable",
							SecurityContext: &corev1.SecurityContext{
								Privileged: func() *bool { b := true; return &b }(),
							},
							Command: []string{"/bin/bash"},

							Args: []string{
								"-c",
								"newcontainer=$(buildah from scratch); buildah add $newcontainer " + kubeletResponse.Items[0] + "; buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=default-counter $newcontainer; buildah commit $newcontainer checkpoint-image:latest; buildah rm $newcontainer; buildah push --creds=myuser:mypasswd --tls-verify=false localhost/checkpoint-image:latest " + registryIp + ":5000/checkpoint-image:latest; while true; do sleep 30; done;",
								// buildah push checkpoint-image to nodeIP:nodePort with username and password, username=myuser and password=mypasswd
								// buildah push --creds=myuser:mypasswd --tls-verify=false localhost/checkpoint-image:latest 10.85.0.8:5000/checkpoint-image:latest
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "checkpointed-image",
									MountPath: "/var/lib/kubelet/checkpoints/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "checkpointed-image",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/checkpoints/",
								},
							},
						},
					},
					NodeName: NodeName,
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Migration{}).
		Complete(r)
}
