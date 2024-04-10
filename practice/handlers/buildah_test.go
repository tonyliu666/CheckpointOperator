package handlers

import (
	"testing"

	"tony123.tw/util"
)

// integration test for buildah getting kubelet client, creating a image for checkpointed pod and push to registry
func TestBuildahPodPushImage(t *testing.T) {
	nodeName := "kubenode02"
	namespace := "docker-registry"
	checkpoint := "/var/lib/kubelet/checkpoints/checkpoint-counter-app-76f6c8d44f-xhhvt_default-counter-2024-04-10T07:02:43Z.tar"

	clientset,err := util.CreateClientSet()
	if err != nil{
		t.Error("unable to get the clientset")
	}

	// fetch the registry IP from kubenode02
	registryIp, err := ReturnRegistryIP(clientset, nodeName)

	err = BuildahPodPushImage(nodeName, namespace, checkpoint, registryIp)

	if err != nil {
		t.Error("unable to push image to registry")
	}

	// check the image has been pushed to registry on kubenode02

	// TODO: check the image alreay existed in the registry, and the image name is checkpoint-image:latest

}
