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
	"os"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"tony123.tw/util"
)

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// delare the logger without initializing it
var logger logr.Logger

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

	logger = log.FromContext(ctx)
	// TODO(user): your logic here
	// detect the docker registry daemonset has received the new image at the timestamp

	// get the pods from namespace docker-registry
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace("docker-registry")); err != nil {
		logger.Error(err, "unable to list pods")
		return ctrl.Result{}, err
	}

	// get the pods whose prefix name is docker-registry
	for _, pod := range pods.Items {
		logger.Info("Pod name", "name", pod.Name)
		// if pod.Name includes docker-registry
		if strings.Contains(pod.Name, "docker-registry") {
			// get the pod ip
			podIP := pod.Status.PodIP
			// curl -X GET <user>:<pass> https://podIP:5000/v2/_catalog

			// create nginx pods
			nginx_pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-pod",
					Namespace: "docker-registry",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			}
			// ssh into that pod
			// kubectl exec -it nginx-pod -- /bin/bash
			ssh_into_pod(nginx_pod.Name, podIP, "docker-registry")

			// create the request with like this curl command:
			// manifest=$(curl -H "Accept: application/vnd.oci.image.manifest.v1+json" http://{podIP}:5000/v2/{image-name}/manifests/latest)

		}

	}
	return ctrl.Result{}, nil
}
func ssh_into_pod(podName string, podIP string, namespace string) {
	clientset, err := util.CreateClientSet()
	if err != nil {
		logger.Error(err, "unable to create clientset")
		return
	}

	// make a curl command in the container
	//curl -H "Accept: application/vnd.oci.image.manifest.v1+json" http://{podIP}:5000/v2/{image-name}/manifests/latest)
	command := []string{"/bin/bash",
		"-c",
		"configDigest=$(curl -H \"Accept: application/vnd.oci.image.manifest.v1+json\" http://" + podIP + ":5000/v2/checkpoint-image/manifests/latest | jq -r '.config.digest')",
		"-c",
		"configBlob=(curl http://", podIP, ":5000/v2/checkpoint-image/blobs/$configDigest)",
		"-c",
		"echo $configBlob",
	}
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, metav1.ParameterCodec)
	kubeconfig := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "unable to get the config")
		return
	}
	
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		logger.Error(err, "unable to create the executor")
		return
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		logger.Error(err, "unable to stream")
		return
	}
}

// SetupWithManager sets up the controller with the Manager.

func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// watch daemonset for docker-registry
		For(&appsv1.DaemonSet{}).
		Complete(r)
}
