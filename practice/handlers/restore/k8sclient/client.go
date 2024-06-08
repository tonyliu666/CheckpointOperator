package k8sclient

import (
	"fmt"
	"os"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

func CreateClientSet() (*kubernetes.Clientset, error) {
	// get the kubernetes config
	config, err := rest.InClusterConfig()

	if err != nil {
		// If running outside the cluster, use kubeconfig file
		fmt.Println("unable to create clientset by service account")
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Log.Error(err, "unable to create clientset")
		return nil, err
	}
	return clientset, nil
}
