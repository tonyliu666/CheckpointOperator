package util

import (
	"strings"
)

func ModifyCheckpointToImageName(checkpoint string) string {
	filePath := checkpoint

	// Step 1: Remove the prefix "/var/lib/kubelet/checkpoints/"
	prefix := "/var/lib/kubelet/checkpoints/"
	remainingPath := strings.TrimPrefix(filePath, prefix)

	parts := strings.SplitN(remainingPath, "_", 2)
	result := parts[0]

	return result
}
