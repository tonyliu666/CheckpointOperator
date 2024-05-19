package util

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNodePortServiceIP(hostIP string,namespace string) (string, error) {
	// Create a new Kubernetes client configuration
	clientset, err := CreateClientSet()
	
	if err != nil {
		return "", err
	}

	// Get the service object with the prefix name docker-registry
	service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "docker-registry-service", metav1.GetOptions{})
	
	if err != nil {
		return "", err
	}
	// return the service domain name
	return fmt.Sprintf("%s:%d", hostIP, service.Spec.Ports[0].NodePort), nil

}
