package handlers

import (
	"encoding/json"
	"io"
	"testing"
)

type kubeletCheckpointResponse struct {
	Items []string `json:"items"`
}

// unit test for GetKubeletClient
func TestGetKubeletClient(t *testing.T) {
	client := GetKubeletClient()
	if client == nil {
		t.Error("unable to get the kubelet client")
	}
}

// integration test for getting the valid http client and checkpointing a pod
func TestCheckpointPod(t *testing.T) {
	// get the pod address for counter pod
	namespace := "default"

	client := GetKubeletClient()
	
	address := "https://192.168.56.3:10250/checkpoint/" + namespace + "/counter-app/counter"

	resp, err := CheckpointPod(client, address)
	if err != nil {
		t.Error("unable to checkpoint the pod")
	}
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


