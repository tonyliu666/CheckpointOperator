package util

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPodHostIP(t *testing.T) {
	client, err := CreateClientSet()
	if err != nil {
		t.Errorf("failed to create clientset: %v", err)
	}
	pod, err := client.CoreV1().Pods("docker-registry").Get(context.TODO(), "docker-registry-78rc6", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get pod: %v", err)
	}
	hostIP, err := GetPodHostIP(pod, "docker-registry")
	if err != nil {
		t.Errorf("failed to get pod host IP: %v", err)
	}
	if hostIP != "192.168.56.5:30010" {
		t.Errorf("unexpected hostIP")
	}

}
