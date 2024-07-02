package handlers

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// integration tests for checkpointing the pod, send the message to kafka broker and migrate the new pod
func TestDeployPodOnNewNode(t *testing.T) {
	// create a new pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-pod",
					Image: "checkpoint-test-pod:latest",
				},
			},
		},
	}

	// deploy the pod on a new node
	err := DeployPodOnNewNode(pod, "default", "new-node")
	if err != nil {
		t.Errorf("DeployPodOnNewNode() error = %v", err)
	}

}
