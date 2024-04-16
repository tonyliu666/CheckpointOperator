package handlers

import (
	"context"
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"tony123.tw/util"
)

// integration test for buildah getting kubelet client, creating a image for checkpointed pod and push to registry
func TestBuildahPodPushImage(t *testing.T) {
	nodeName := "kubenode02"
	namespace := "docker-registry"
	checkpoint := "/var/lib/kubelet/checkpoints/checkpoint-counter-app-76f6c8d44f-xhhvt_default-counter-2024-04-10T07:02:43Z.tar"

	clientset, err := util.CreateClientSet()
	if err != nil {
		t.Error("unable to get the clientset")
	}

	// fetch the registry IP from kubenode02
	registryIp, err := ReturnRegistryIP(clientset, nodeName)

	if err != nil {
		t.Error("unable to get the registry ip")
	}

	err = BuildahPodPushImage(nodeName, namespace, checkpoint, registryIp)

	if err != nil {
		t.Error("unable to push image to registry")
	}

	// check the image has been pushed to registry on kubenode02

	// TODO: check the image alreay existed in the registry, and the image name is checkpoint-image:latest

	// check the image whose name is checkpoint-image has been pushed to registry on kubenode02
	// get the docker registry pod on kubenode02

	registryPodList, err := clientset.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=docker-registry",
	})
	if err != nil {
		t.Error("unable to get the registry pod")
	}
	// find the registry pod which is on the same node as the pod
	var registryPod v1.Pod
	for _, registryPod = range registryPodList.Items {
		if registryPod.Spec.NodeName == nodeName {
			break
		}
	}

	// check the folder, /var/lib/registry/docker/registry/v2/repositories has the folder whose name is checkpoint-image
	// get into the container of the registry pod
	// use codes to check that folder exists in the registry pod, just like the same as entering "kubectl exec -ti registry-pod-name -- /bin/bash -c 'ls /var/lib/registry/docker/registry/v2/repositories'"

	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(registryPod.Name).Namespace("docker-registry").SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: registryPod.Spec.Containers[0].Name,
		Command: []string{
			"/bin/bash",
			"-c",
			"ls /var/lib/registry/docker/registry/v2/repositories/checkpoint-image",
		},
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		TTY:    true,
	}, metav1.ParameterCodec)

	kubeconfig := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	exec, err := remotecommand.NewSPDYExecutor(
		config,
		"POST",
		req.URL(),
	)
	if err != nil {
		t.Error("unable to create executor")
	}
	exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})

}
