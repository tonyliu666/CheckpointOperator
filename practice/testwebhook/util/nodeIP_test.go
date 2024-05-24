package util

import (
	"context"
	"os"
	"testing"
	"testwebhook/k8sclient"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetNodeNameByHostIP(t *testing.T) {
	client, err := k8sclient.CreateClientSet()
	if err != nil {
		t.Errorf("unable to create clientset")
	}

	// list the docker registry pods in docker-registry namespace
	podList, err := client.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=docker-registry",
	})
	if err != nil {
		t.Errorf("unable to list the pods in the docker-registry namespace")
	}
	podIP := "10.244.96.50"
	nodeName, err := GetNodeNameByHostIP(podIP, podList)

	// list the pods in the docker-registry namespace

	if err != nil {
		t.Errorf("failed to get the node name by host IP: %s", podIP)
	}
	if nodeName != "kubenode02" {
		t.Errorf("the node name is not correct")
	}

}

func createClientSet() (*kubernetes.Clientset, error) {
	// get the kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If running outside the cluster, use kubeconfig file
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "unable to create clientset")
		return nil, err
	}
	return clientset, nil
}
