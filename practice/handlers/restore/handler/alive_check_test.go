package handler

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestAliveCheck(t *testing.T) {
	nameSpace := "default"
	newPodName := "checkpoint-redis1"
	nodeName := "kubenode03"
	alive, state := AliveCheck(nameSpace, newPodName, nodeName)
	if !alive {
		t.Error("Alive check failed")
	}
	expectedState := v1.PodPhase("Running")

	if state != expectedState {
		t.Errorf("Expected state to be Running, got %s", state)
	}
}
