package handlers

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	util "tony123.tw/util"
)

func TestBuildahPodPushImage(t *testing.T) {
	// integration test for buildah getting kubelet client, creating a image for checkpointed pod and push to registry
	kubeletResponse := &KubeletCheckpointResponse{}
	// make a checkpoint for the pod on default namespace
	err := kubeletResponse.RandomCheckpointPod("default")
	if err != nil {
		t.Error("error while checkpointing pod: ", "error", err)
	}

	namespace := "docker-registry"
	// checkpoint := "/var/lib/kubelet/checkpoints/checkpoint-counter-app-76f6c8d44f-xhhvt_default-counter-2024-04-10T07:02:43Z.tar"
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println("Failed to get Kubernetes configuration:", err)
		return
	}

	// Create a new Kubernetes client
	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		fmt.Println("Failed to create Kubernetes client:", err)
		return
	}

	// Define the label selector
	labelSelector := client.MatchingLabels{"app": "docker-registry"}

	// Create a PodList object to store the results
	pods := &corev1.PodList{}

	// List the pods with the label app=docker-registry
	err = k8sClient.List(context.Background(), pods, labelSelector)
	if err != nil {
		fmt.Println("Failed to list pods:", err)
		return
	}

	clientset, err := util.CreateClientSet()
	if err != nil {
		t.Error("unable to get the clientset")
	}

	random := rand.Intn(len(pods.Items))
	nodeName := pods.Items[random].Spec.NodeName
	// randomly choose a pod from the pods list
	registryIp, err := ReturnRegistryIP(clientset, nodeName)

	if err != nil {
		t.Error("unable to get the registry ip")
	}

	err = BuildahPodPushImage(0, nodeName, namespace, kubeletResponse.Items[0], registryIp)

	if err != nil {
		t.Error("unable to push image to registry")
	}

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

	// check the registry folder, /var/lib/registry/docker/registry/v2/repositories has the folder whose name is checkpoint-image
	// get into the container of the registry pod
	// use codes to check that folder exists in the registry pod, just like the same as entering "kubectl exec -ti registry-pod-name -- /bin/bash -c 'ls /var/lib/registry/docker/registry/v2/repositories'"

	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(registryPod.Name).Namespace("docker-registry").SubResource("exec")
	podName := util.ModifyCheckpointToImageName(kubeletResponse.Items[0])
	req.VersionedParams(&v1.PodExecOptions{
		Container: registryPod.Spec.Containers[0].Name,
		Command: []string{
			"/bin/bash",
			"-c",
			"ls /var/lib/registry/docker/registry/v2/repositories/" + podName,
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

	// TODO: check the status of the buildah pod is completed than delete the buildah pod
	// get the buildah pod
	now := time.Now()
	for {
		// check every 5 seconds if the buildah pod is completed
		time.Sleep(5 * time.Second)
		buildahPodList, _ := clientset.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app=buildah",
		})
		if len(buildahPodList.Items) == 0 {
			break
		}

		if time.Since(now) > 30*time.Second {
			t.Error("timeout")
			break
		}
	}
}
