package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GetPodHostIP(pod *corev1.Pod, namespace string) (string, error) {
	// Get the container hostPort
	hostPort := ""
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			hostPort = fmt.Sprintf("%d", port.HostPort)
			break 
		}
	}
	// get the hostIP of the pod
	hostIP := pod.Status.HostIP
	// return the service domain name
	return fmt.Sprintf("%s:%s", hostIP, hostPort), nil
}
