package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	util "tony123.tw/util"
)

// isImageNotFoundError checks if the error is due to the image not being found
func isImageNotFoundError(err error) bool {
	// Check if the error message or type indicates that the image is not found
	// This is an example and may need to be adjusted based on the actual error message and type
	return strings.Contains(err.Error(), "ErrImagePull") || strings.Contains(err.Error(), "ImagePullBackOff")
}

func OriginalImageChecker(pod *corev1.Pod, dstNode string) error {
	imageIDList := []string{}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		imageID := containerStatus.ImageID
		imageIDList = append(imageIDList, imageID)
	}

	// check the image id of the original pod on the destination node
	err := util.CheckImageIDExistOnNode(imageIDList, dstNode)
	if err != nil {
		log.Log.Error(err, "unable to check image id")
		return fmt.Errorf("unable to check image id: %w", err)
	}
	// check the check-image-id job is finished or not
	// set the context for the time limit of the job

	err = util.CheckJobStatus("check-image-id", "Succeeded")
	return nil
}

func DeployPodOnNewNode() (string,error){
	// deploy a new pod on the destination node
	if len(util.CheckpointPodName) == 0 {
		log.Log.Info("No more pods to deploy")
		return "",fmt.Errorf("no more pods to deploy")
	}
	podName := util.CheckpointPodName[0]
	// pop the first element of the list
	util.CheckpointPodName = util.CheckpointPodName[1:]
	migratePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "checkpoint-" + podName, 
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "checkpoint-" + podName, 
					Image: "localhost/" + "checkpoint-" + podName + ":latest",
				},
			},
			NodeName: util.DestinationNode,
		},
	}

	clientset, err := util.CreateClientSet()
	if err != nil {
		log.Log.Error(err, "unable to create clientset", err)
		return "",fmt.Errorf("unable to create clientset: %w", err)
	}

	// TODO: create a pod on the destination node when the checkpoint image has been pushed to the destination node
	newpod, err := clientset.CoreV1().Pods("default").Create(context.Background(), migratePod, metav1.CreateOptions{})
	if err != nil {
		if isImageNotFoundError(err) {
			log.Log.Error(err, "Image not found, retrying to create the pod")
			// Optionally, you can add a delay before retrying
			time.Sleep(5 * time.Second)
			newpod, err = clientset.CoreV1().Pods("default").Create(context.Background(), migratePod, metav1.CreateOptions{})
			if err != nil {
				log.Log.Error(err, "Failed to create the pod after retry")
				return "", err
			}
		} else {
			log.Log.Error(err, "Failed to create the pod")
			return "",err
		}
	}
	log.Log.Info("Pod created successfully", "podName", newpod.Name)

	return podName,nil
}
