package util

import (
	"os"

	log "github.com/sirupsen/logrus" // Add the missing import for logrus
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateClientSet() (*kubernetes.Clientset, error) {
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
