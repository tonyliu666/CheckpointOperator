package handlers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
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
		
		// TODO: remove the pod from the ProcessPodsMap
		// oldPodName is the key that remove the checkpoint- prefix from the pod name
		oldPodName := strings.TrimPrefix(podName, "checkpoint-")
		info, ok := util.ProcessPodsMap[oldPodName].(util.MigrationInfo)
		
		if !ok {
			log.Log.Error(err, "unable to get the information of the pod")
			return fmt.Errorf("unable to get the information of the pod: %w", err)
		}
		// remove the pod from the ProcessPodsMap
		delete(util.ProcessPodsMap, oldPodName)
		log.Log.Info("Pod removed from the ProcessPodsMap", "podName", util.ProcessPodsMap)

		imageName := podName + ":latest"
		podIP, err := util.GetPodHostPort(pod, "docker-registry")
		if err != nil {
			return fmt.Errorf("can't get nodePort IP: %w", err)
		}
		
		imageLocation := fmt.Sprintf("%s/%s", podIP, imageName)
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
		

		// make sure the destantion namespace is created
		err = checkDestinationNameSpaceExists(info.DestinationNamespace)
		if err != nil {
			log.Log.Error(err, "unable to check destination namespace")
			return fmt.Errorf("unable to check destination namespace: %w", err)
		}

		newpod, err := clientset.CoreV1().Pods(info.DestinationNamespace).Create(context.TODO(), migratePod, metav1.CreateOptions{})
		if err != nil {
			log.Log.Error(err, "unable to deploy new pod")
			return fmt.Errorf("unable to create pod: %w", err)
		}
		// remove the pod from the ProcessPodsMap
		
		if err := ProduceMessageToDifferentTopics(newpod.Name, info.SourceNamespace, nodeName); err != nil {
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
func checkDestinationNameSpaceExists(destination string) error {
	clientset, err := util.CreateClientSet()
	if err != nil {
		log.Log.Error(err, "unable to create clientset")
		return fmt.Errorf("unable to create clientset: %w", err)
	}
	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), destination, metav1.GetOptions{})
	if err != nil {
		log.Log.Error(err, "unable to get the destination namespace")
		// create the namespace
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: destination,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Log.Error(err, "unable to create the destination namespace")
			return fmt.Errorf("unable to create the destination namespace: %w", err)
		}
	}
	return nil
}
