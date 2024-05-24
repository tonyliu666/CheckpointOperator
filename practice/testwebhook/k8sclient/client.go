package k8sclient

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	log "sigs.k8s.io/controller-runtime/pkg/log"
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
func ListPods(namespace string,selector string) (*v1.PodList, error) {
	client, err := CreateClientSet()
	if err != nil {
		fmt.Println("unable to create clientset")
		return nil,err
	}
	// TODO: get the docker registry pod list and the pod deployed on which node
	registryPodList, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		fmt.Println("unable to list the pods in the docker-registry namespace", err)
		return nil, err
	}
	return registryPodList, nil
}
