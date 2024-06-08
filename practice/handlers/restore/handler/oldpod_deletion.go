package handler

import (
	"context"
	"restore-daemon/k8sclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "github.com/sirupsen/logrus"
)

func DeleteOldPod(nameSpace string, oldPodName string) bool{
	// get the message from kafka
	clientset, err := k8sclient.CreateClientSet()
	if err != nil {
		log.Error ("unable to create the clientset")
	}
	// check the old pod is deleted or not
	_, err = clientset.CoreV1().Pods(nameSpace).Get(context.TODO(), oldPodName, metav1.GetOptions{})
	if err != nil {
		if err.Error() == "pods \"" + oldPodName + "\" not found" {
			log.Info("The old pod has been deleted successfully")
			return true
		}
		return false
	}
	// delete the old pod
	err = clientset.CoreV1().Pods(nameSpace).Delete(context.TODO(), oldPodName, metav1.DeleteOptions{})
	if err != nil {
		log.Error(err, "unable to delete the old pod")	
	}
	log.Info("The old pod is deleted successfully")
	return true
}