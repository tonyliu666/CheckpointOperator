package util

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func CheckPodStatus(originPodName string, state string, namespace string) error {
	clientset, err := CreateClientSet()
	counter := 0
	if err != nil {
		return fmt.Errorf("unable to create clientset: %w", err)
	}
	log.Log.Info("origin pod name", "originPodName", originPodName)
	for {
		// check pod status is running within 30 seconds
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		// check the buildah pod in migration namespace whose prefix of pod name is buildah-job
		if err != nil {
			log.Log.Error(err, "unable to list the pods in migration namespace")
			return fmt.Errorf("unable to get pod: %w", err)
		}

		for _, pod := range pods.Items {
			log.Log.Info("pod status", "pod", pod.Name, "status", pod.Status.Phase)
			// check the suffix of pod name is the same as the origin pod name, ex: buildah-job-nginx and originPodName is nginx
			if strings.Contains(pod.Name, originPodName) && string(pod.Status.Phase) == state {
				log.Log.Info("pod is in in state", "pod", pod.Name)
				return nil
			}
		}
		time.Sleep(10 * time.Second)
		counter++
		if counter==300{
			log.Log.Info("pod is not in state", "state", state)
			return fmt.Errorf("pod is not in %s state, timeout", state)
		}
	}
}
