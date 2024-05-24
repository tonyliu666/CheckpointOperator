package k8sclient

import "testing"


func TestListPods(t *testing.T) {
	namespace := "docker-registry"
	selector := "app=docker-registry"
	// list the docker registry pods in docker-registry namespace
	podList, err := ListPods(namespace, selector)
	if err != nil {
		t.Errorf("unable to list the pods in the docker-registry namespace")
	}
	if len(podList.Items) == 0 {
		t.Errorf("no pods found in the docker-registry namespace")
	}
}
