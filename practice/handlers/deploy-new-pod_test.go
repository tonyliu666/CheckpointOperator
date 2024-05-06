package handlers

import (
	"sync"
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

func TestDeployPodOnNewNode(t *testing.T) {
	// get the pods whose label is app: docker-registry
	pods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "docker-registry-h9zpq",
				},
				Spec: corev1.PodSpec{
					NodeName: "kubenode03",
				},
			},
		},
	}

	// depend on how many docker registry pods, create how many go routines
	var wg sync.WaitGroup

	err := DeployPodOnNewNode(&pods.Items[0], &wg)
	if err != nil {
		t.Errorf("DeployPodOnNewNode failed: %v", err)
	}
}
