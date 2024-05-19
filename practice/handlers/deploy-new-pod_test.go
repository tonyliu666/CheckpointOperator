package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// integration tests for checkpointing the pod, send the message to kafka broker and migrate the new pod
func TestDeployPodOnNewNode(t *testing.T) {
	// get the pods whose label is app: docker-registry
	k := KubeletCheckpointResponse{}
	err, nodeName := k.randomCheckpointPodReturnNodeName("default")

	// produce the message
	err = ProduceMessage(nodeName, k.Items[0])
	if err != nil {
		t.Errorf("ProduceMessage failed: %v", err)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println("Failed to get Kubernetes configuration:", err)
		return
	}
	// Create a new Kubernetes client
	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		fmt.Println("Failed to create Kubernetes client:", err)
		return
	}

	// Define the label selector
	labelSelector := client.MatchingLabels{"app": "docker-registry"}

	// Create a PodList object to store the results
	pods := &corev1.PodList{}

	// List the pods with the label app=docker-registry
	err = k8sClient.List(context.Background(), pods, labelSelector)
	if err != nil {
		fmt.Println("Failed to list pods:", err)
		return
	}
	// find which docker registry pod whose node name is nodeName
	var pod corev1.Pod
	for _, pod = range pods.Items {
		if pod.Spec.NodeName == nodeName {
			break
		}
	}

	// depend on how many docker registry pods, create how many go routines

	err = DeployPodOnNewNode(&pod)
	if err != nil {
		t.Errorf("DeployPodOnNewNode failed: %v", err)
	}
}

// the utility for test
func (k *KubeletCheckpointResponse) randomCheckpointPodReturnNodeName(namespace string) (error, string) {
	cfg, err := config.GetConfig()
	if err != nil {
		return err, ""
	}

	// Create a new Kubernetes client using the configuration
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		return err, ""
	}

	pods := &corev1.PodList{}
	if err := client.List(context.Background(), pods, cli.InNamespace(namespace)); err != nil {
		return err, ""
	}
	// get the nodeIP of the pod
	random := rand.Intn(len(pods.Items))
	nodeIP := pods.Items[random].Status.HostIP
	// get the container name of the pod
	containerName := pods.Items[random].Spec.Containers[0].Name
	address := "https://" + nodeIP + ":10250/checkpoint/" + namespace + "/" + pods.Items[random].Name + "/" + containerName

	httpClient := GetKubeletClient()
	resp, err := CheckpointPod(httpClient, address)
	if err != nil {
		return err, ""
	}

	if err != nil {
		return err, ""
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err, ""
	}

	err = json.Unmarshal(body, k)
	if err != nil {
		return err, ""
	}
	return nil, pods.Items[random].Spec.NodeName
}
