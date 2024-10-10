package util

import (
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
)

func GetNodeNameByHostIP(podIP string, podList *corev1.PodList) (string, error) {
	// find the node name which is on the same node as the pod whose IP is podIP
	var nodeName string
	for _, pod := range podList.Items {
		if pod.Status.PodIP == podIP {
			nodeName = pod.Spec.NodeName
			log.Println("found the nodeName", nodeName)
			break
		}
	}

	if nodeName == "" {
		return "", fmt.Errorf("can't find the node name by host IP: %s ", podIP)
	}
	return nodeName, nil
}
