package handlers

import "testing"

func TestProduceMessage(t *testing.T) {
	checkpointFileName := "checkpoint-counter-app-76f6c8d44f-mrf96_default-counter-2024-04-10T07:56:14Z"
	nodeName := "kubenode01"

	ProduceMessage(checkpointFileName, nodeName)

}
