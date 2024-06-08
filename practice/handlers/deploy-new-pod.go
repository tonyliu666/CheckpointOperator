package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	util "tony123.tw/util"
)

func DeployPodOnNewNode(pod *corev1.Pod) error {
	msgList, err := ConsumeMessage(pod.Spec.NodeName)
	if err != nil {
		return fmt.Errorf("failed to consume message: %w", err)
	}

	for _, msg := range msgList {
		nodeName := string(msg.Key)
		podName := string(msg.Value)
		podIP := pod.Status.PodIP
		registryAPIAddr := fmt.Sprintf("http://%s:5000/v2/_catalog", podIP)
		rsp, err := http.Get(registryAPIAddr)
		if err != nil {
			return fmt.Errorf("failed to get registry API address: %w", err)
		}
		defer rsp.Body.Close()

		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		log.Log.Info("checkPointPodName: ", "podName", podName, "body", string(body))
		// TODO: If buildah pushing a big chunk of image to docker registry, that body will not contain the podname at this moment
		if strings.Contains(string(body), podName) {
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
				return fmt.Errorf("unable to create clientset: %w", err)
			}
			// TODO: replace default namespace with the namespace of the pod
			newpod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), migratePod, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("unable to create pod: %w", err)
			}
			// TODO: notify the other service to watch this new pod created successfully or not
			// send kafka message to broker, default namesapce in the later will be changed to random namespace
			if err := ProduceMessageToDifferentTopics(newpod.Name, "default", nodeName); err != nil {
				return fmt.Errorf("failed to produce message: %w", err)
			}
			log.Log.Info("Pod created",
				"podName", newpod.Name,
				"nodeName", nodeName,
			)
		}
	}
	return nil
}
