package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func createClientSet() (*kubernetes.Clientset, dynamic.Interface, error) {
	// Get the Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If running outside the cluster, use kubeconfig file
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, err
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Unable to create clientset: ", err)
		return nil, nil, err
	}
	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Error("Unable to create dynamic client: ", err)
		return nil, nil, err
	}
	return clientset, dynamicClient, nil
}

func main() {
	// Create Clientset
	clientset, dynamicClient, err := createClientSet()
	if err != nil {
		log.Error("Failed to create clientset: ", err)
		return
	}

	// get the events every 3 minutes
	for {
		// List events in the default namespace
		events, err := clientset.CoreV1().Events("practice-system").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error("Failed to list events: ", err)
			return
		}

		// Define the custom resource
		gvr := schema.GroupVersionResource{
			Group:    "api.my.domain",
			Version:  "v1alpha1",
			Resource: "migrations",
		}

		// Watch for events
		watch, err := dynamicClient.Resource(gvr).Namespace("practice-system").Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error("Failed to watch custom resources: ", err)
			return
		}

		// Print event details
		fmt.Println("Listing events in the default namespace:")
		for _, event := range events.Items {
			fmt.Printf("Name: %s, Namespace: %s, Reason: %s, Message: %s, CreationTimestamp: %s\n",
				event.Name, event.Namespace, event.Reason, event.Message, event.CreationTimestamp)
		}
		

		fmt.Println("Watching for events related to Migration resources...")
		for event := range watch.ResultChan() {
			fmt.Println("Event: ", event.Type)
			fmt.Println("Object: ", event.Object)
		}
		time.Sleep(3 * time.Minute)
	}
}
