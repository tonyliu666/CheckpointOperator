package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func createClientSet() (*kubernetes.Clientset, error) {
	// Get the Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If running outside the cluster, use kubeconfig file
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Unable to create clientset: ", err)
		return nil, err
	}

	return clientset, nil
}

func main() {
	// Create Clientset
	clientset, err := createClientSet()
	if err != nil {
		log.Error("Failed to create clientset: ", err)
		return
	}

	// get the events every 3 minutes
	for {
		// List events in the default namespace
		events, err := clientset.CoreV1().Events("docker-registry").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error("Failed to list events: ", err)
			return
		}

		// Print event details
		fmt.Println("Listing events in the default namespace:")
		for _, event := range events.Items {
			fmt.Printf("Name: %s, Namespace: %s, Reason: %s, Message: %s, CreationTimestamp: %s\n",
				event.Name, event.Namespace, event.Reason, event.Message, event.CreationTimestamp)
		}

		time.Sleep(3 * time.Minute)
	}
}
