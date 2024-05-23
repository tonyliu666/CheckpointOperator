package util

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	fake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetNodeNameByHostIP(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	// list the docker registry pods in docker-registry namespace
	registryPodList := &corev1.PodList{}
	if err := fakeClient.List(context.TODO(), registryPodList, client.InNamespace("docker-registry"), client.MatchingLabels{"app": "docker-registry"}); err != nil {
		t.Errorf("unable to list pods")
	}

	// list the pods in the docker-registry namespace
	podIP := "10.244.96.42"
	for _, pod := range registryPodList.Items {
		if pod.Status.PodIP == podIP {
			nodeName := pod.Spec.NodeName
			if nodeName != "kubenode2" {
				t.Errorf("the node name is not correct")
			}
		}
	}
}
