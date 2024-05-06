package handlers

import (
	"context"
	"encoding/json"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

type kubeletCheckpointResponse struct {
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

	cfg, err := config.GetConfig()
	if err != nil {
		t.Error(err)
		return
	}

	// Create a new Kubernetes client using the configuration
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Error("unable to create the client")
		return
	}

	pods := &corev1.PodList{}
	if err := client.List(context.Background(), pods, cli.InNamespace(namespace)); err != nil {
		t.Error("unable to list the pods")
	}
	// get the nodeIP of the pod
	nodeIP := pods.Items[0].Status.HostIP
	address := "https://" + nodeIP + ":10250/checkpoint/" + namespace + "/" + pods.Items[0].Name + "/counter"

	httpClient := GetKubeletClient()
	resp, err := CheckpointPod(httpClient, address)

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error("error while reading body: ", "error", err)
	}
	kubeletResponse := &kubeletCheckpointResponse{}
	err = json.Unmarshal(body, kubeletResponse)
	if err != nil {
		t.Error("Error unmarshalling kubelet response")
	}
}
