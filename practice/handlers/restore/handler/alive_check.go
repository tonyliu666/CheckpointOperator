// this is for checking the new pod is alive on new node or not

package handler

import (
	"context"
	"restore-daemon/k8sclient"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "github.com/sirupsen/logrus"
)

func AliveCheck(nameSpace string, newPodName string, nodeName string) (bool, v1.PodPhase) {
	// get the message from kafka
	clientset, err := k8sclient.CreateClientSet()
	if err != nil {
		log.Error("unable to create the clientset")
	}
	// get that pod by its name and namespace
	pod, err := clientset.CoreV1().Pods(nameSpace).Get(context.TODO(), newPodName, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "unable to get the pod")
	}
	expectedState := v1.PodPhase("Running")
	// check the pod is alive or not
	if pod.Status.Phase == expectedState {
		log.Info("The pod is alive on the ", nodeName, " node")
		return true, expectedState
	}
	return false, pod.Status.Phase
}
