// this is for checking the new pod is alive on new node or not

package restore

import (
	"context"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	util "tony123.tw/util"
)

func AliveCheck(nameSpace string, newPodName string, nodeName string) (bool, v1.PodPhase) {
	// get the message from kafka
	clientset, err := util.CreateClientSet()
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
