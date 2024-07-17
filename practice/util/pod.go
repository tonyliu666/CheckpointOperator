package util

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeletePod(podName string) error {
	clientset, err := CreateClientSet()
	if err != nil {
		return fmt.Errorf("unable to create clientset: %w", err)
	}
	err = clientset.CoreV1().Pods("default").Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("unable to delete pod: %w", err)
	}
	return nil
}
func CheckPodStatus(podName string, state string, checkingTime int) error {
	clientset, err := CreateClientSet()
	if err != nil {
		return fmt.Errorf("unable to create clientset: %w", err)
	}
	now := time.Now()
	for {
		// check pod status is running within 30 seconds
		if time.Since(now) > time.Duration(checkingTime)*time.Second {
			return fmt.Errorf("pod is not in running state within 30 seconds")
		}
		pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to get pod: %w", err)
		}
		if string(pod.Status.Phase) == state {
			break
		}
	}
	return nil
}
