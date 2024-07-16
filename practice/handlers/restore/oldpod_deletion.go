package restore

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "github.com/sirupsen/logrus"
	util "tony123.tw/util"
)

func DeleteOldPod(nameSpace string, oldPodName string) error{
	// get the message from kafka
	clientset, err := util.CreateClientSet()
	if err != nil {
		log.Error ("unable to create the clientset")
		return err
	}
	// check the old pod is deleted or not
	_, err = clientset.CoreV1().Pods(nameSpace).Get(context.TODO(), oldPodName, metav1.GetOptions{})
	if err != nil {
		if err.Error() == "pods \"" + oldPodName + "\" not found" {
			log.Info("The old pod has been deleted successfully")
			return nil
		}
		return err
	}
	// delete the old pod
	err = clientset.CoreV1().Pods(nameSpace).Delete(context.TODO(), oldPodName, metav1.DeleteOptions{})
	if err != nil {
		log.Error(err, "unable to delete the old pod")
		return err	
	}
	log.Info("The old pod is deleted successfully")
	return nil
}