package handlers

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type KubeletCheckpointResponse struct {
	Items []string `json:"items"`
}
type MigrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// unit test for GetKubeletClient
func TestGetKubeletClient(t *testing.T) {
	client := GetKubeletClient()
	if client == nil {
		t.Error("unable to get the kubelet client")
	}
}

// integration test for getting the valid http client and checkpointing a pod
// the checkpointed pod is the counter pod located at the default namespace
func TestCheckpointPod(t *testing.T) {
	// get the pod address for counter pod
	namespace := "default"
	kubeletResponse := &KubeletCheckpointResponse{}
	err := kubeletResponse.RandomCheckpointPod(namespace)
	if err != nil {
		t.Error("Error unmarshalling kubelet response")
	}
}

// the utility for test
func (k *KubeletCheckpointResponse) RandomCheckpointPod(namespace string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// Create a new Kubernetes client using the configuration
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		return err
	}

	pods := &corev1.PodList{}
	if err := client.List(context.Background(), pods, cli.InNamespace(namespace)); err != nil {
		return err
	}
	// get the nodeIP of the pod
	random := rand.Intn(len(pods.Items))
	nodeIP := pods.Items[random].Status.HostIP
	// get the container name of the pod
	containerName := pods.Items[random].Spec.Containers[0].Name
	address := "https://" + nodeIP + ":10250/checkpoint/" + namespace + "/" + pods.Items[random].Name + "/" + containerName

	httpClient := GetKubeletClient()
	resp, err := CheckpointPod(httpClient, address)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, k)
	if err != nil {
		return err
	}
	return nil

}
