package util

import (
	"testing"
)

func TestModifyCheckpointToImageName(t *testing.T) {
	checkpoint := "/var/lib/kubelet/checkpoints/checkpoint-nginx-deployment-85bfd45d84-v4mlg_default-nginx-2024-05-06T13:38:10Z.tar"
	result := ModifyCheckpointToImageName(checkpoint)
	if result != "checkpoint-nginx-deployment-85bfd45d84-v4mlg" {
		t.Error("Expected checkpoint-nginx-deployment but got ", result)
	}
}
