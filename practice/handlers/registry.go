package handlers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ReturnRegistryIP(clientset *kubernetes.Clientset, nodeName string) (string, error) {
	// find the registry pod which is on the same node as the pod
	registryPodList, err := clientset.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=docker-registry",
	})
	if err != nil {
		return "", err
	}
	// find the registry pod which is on the same node as the pod
	var registryPod v1.Pod
	for _, registryPod = range registryPodList.Items {
		if registryPod.Spec.NodeName == nodeName {
			break
		}
	}
	return registryPod.Status.PodIP, nil
}
