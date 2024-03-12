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

	// TODO(user): your logic here
	// make install before running the operator, because for api perspective, they don't understand the CRD
	// migration := &apiv1alpha1.Migration{}
	// pod := &corev1.Pod{}

	// loop over the pods in default namespace and find the pods with the postgresql image
	// Get the client
	podList := &corev1.PodList{}
	// get the migration object
	migration := &apiv1alpha1.Migration{}
	err := r.Get(ctx, req.NamespacedName, migration)
	if err != nil {
		l.Error(err, "unable to fetch the migration object")
		return ctrl.Result{}, err
	}
	err = r.List(ctx, podList)
	if err != nil {
		l.Error(err, "unable to list pods")
		return ctrl.Result{}, err
	}
	// loop over the pods
	for _, pod := range podList.Items {
		// loop over the containers in the pod
		// check which node the pod is running on

		// get the node which the pod is running on
		// ssh into the node

		// call the kubelet checkpoint api
		// curl -X POST "https://localhost:10250/checkpoint/default/counters/counter" --insecure --cert /etc/kubernetes/pki/apiserver-kubelet-client.crt --cacert /etc/kubernetes/pki/ca.crt --key /etc/kubernetes/pki/apiserver-kubelet-client.key

		if pod.Status.Phase == corev1.PodRunning && pod.Name == migration.Spec.PodName {

			for _, container := range pod.Spec.Containers {
				// call the container checkpoint api
				//  curl -X POST "https://localhost:10250/checkpoint/default/counters/counter" --insecure --cert /etc/kubernetes/pki/apiserver-kubelet-client.crt --cacert /etc/kubernetes/pki/ca.crt --key /etc/kubernetes/pki/apiserver-kubelet-client.key

				address := fmt.Sprintf(
					"https://%s:%d/checkpoint/%s/%s/%s",
					// util.KubeletAddress[pod.HostIp],
					"192.168.56.3",
					// util.KubeletPorts[pod.HostIp],
					10250,
					migration.Spec.Namespace,
					migration.Spec.PodName,
					container.Name,
				)
				// ssh into the node and run the command

				// call the kubelet checkpoint api
				// l.Info("checkpoint kubelet", "address", address)

				l.Info("checkpoint kubelet", "address", address)
				client := handler.GetKubeletClient()
				if err != nil {
					l.Error(err, "unable to create a new request")
					return ctrl.Result{}, err
				}
				CheckpointStartTime := time.Now()
				resp, err := client.Post(address, "application/json", strings.NewReader(""))
				CheckpointEndTime := time.Now()
				CheckpointDuration := CheckpointEndTime.Sub(CheckpointStartTime).Milliseconds()
				log.Log.Info("Checkpoint Duration: ", "Duration", CheckpointDuration)
				// err now is facing the problem that the status code is 401 unauthorized
				if err != nil {
					l.Error(err, "unable to send the request")
					return ctrl.Result{}, err
				}
				if resp.StatusCode >= 300 || resp.StatusCode < 200 {
					log.Log.Info("http post returned: ", "code", resp.StatusCode, "status", resp.Status, "body")
					continue
				}
				// close the response body
				defer resp.Body.Close()
				// check the response status code
				if resp.StatusCode != http.StatusOK {
					l.Info("the response status code is not 200")
					return ctrl.Result{}, nil
				}

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

				// Create a new deployment with image buildah
				// docker run -–privileged -v ./image/:/image -ti quay.io/buildah/stable /bin/bash
				// bind mount the checkpointed image to the buildah container
				num := int32(1)

				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "buildah-deployment",
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
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										// privileged container
										Name:  "buildah",
										Image: "quay.io/buildah/stable",
										Command: []string{
											"/bin/bash",
										},
										Args: []string{
											"-c",
											"buildah bud -t buildah-image /image",
										},
										// bind mount the checkpointed image /var/lib/kubelet/checkpoints/ to /image
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "checkpointed-image",
												MountPath: "/image",
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
							},
						},
					},
				}

				createdDeployment, err := clientset.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
				if err != nil {
					panic(err.Error())
				}
				fmt.Printf("Deployment %q created\n", createdDeployment.GetObjectMeta().GetName())
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Migration{}).
		Complete(r)
}
