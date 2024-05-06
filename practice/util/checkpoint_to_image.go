package util

import (
	"strings"
)

func ModifyCheckpointToImageName(checkpoint string) string {
	filePath := "/var/lib/kubelet/checkpoints/checkpoint-nginx-deployment-85bfd45d84-v4mlg_default-nginx-2024-05-06T13:38:10Z.tar"

	// Step 1: Remove the prefix "/var/lib/kubelet/checkpoints/"
	prefix := "/var/lib/kubelet/checkpoints/"
	remainingPath := strings.TrimPrefix(filePath, prefix)

	parts := strings.SplitN(remainingPath, "_", 2)
	result := parts[0]

	return result
}
