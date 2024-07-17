package handlers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	util "tony123.tw/util"
)
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

func DeployPodOnNewNode(pod *corev1.Pod) error {
	msgList, err := ConsumeMessage(pod.Spec.NodeName)
	if err != nil {
		return fmt.Errorf("failed to consume message: %w", err)
	}

	for _, msg := range msgList {
		nodeName := string(msg.Key)
		podName := string(msg.Value)

		// TODO: If buildah pushing a big chunk of image to docker registry, that body will not contain the podname at this moment

		imageName := podName + ":latest"
		registryServiceIP, err := util.GetPodHostIP(pod, "docker-registry")
		if err != nil {
			return fmt.Errorf("can't get nodePort IP: %w", err)
		}
		imageLocation := fmt.Sprintf("%s/%s", registryServiceIP, imageName)
		migratePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  podName,
						Image: imageLocation,
					},
				},
				NodeName: nodeName,
			},
		}

		clientset, err := util.CreateClientSet()
		if err != nil {
			log.Log.Error(err, "unable to create clientset", err)
			return fmt.Errorf("unable to create clientset: %w", err)
		}
		// TODO: replace default namespace with the namespace of the pod
		newpod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), migratePod, metav1.CreateOptions{})
		if err != nil {
			log.Log.Error(err, "unable to deploy new pod")
			return fmt.Errorf("unable to create pod: %w", err)
		}
		// TODO: notify the other service to watch this new pod created successfully or not
		// send kafka message to broker, default namesapce in the later will be changed to random namespace
		if err := ProduceMessageToDifferentTopics(newpod.Name, "default", nodeName); err != nil {
			log.Log.Error(err, "failed to produce different message")
			return fmt.Errorf("failed to produce message: %w", err)
		}
		log.Log.Info("Pod created",
			"podName", newpod.Name,
			"nodeName", nodeName,
		)

	}
	return nil
}
