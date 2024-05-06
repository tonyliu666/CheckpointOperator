package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"unicode"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"tony123.tw/util"
)

func DeployPodOnNewNode(pod *corev1.Pod, wg *sync.WaitGroup) error {
	// get the message from kafka broker
	// TODO: Consume the message from the Kafka broker with the real go codes
	// I will containerize these go codes later
	msgList, err := ConsumeMessage(pod.Spec.NodeName)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Ready to comsume message")
	// TODO: Deploy a new pod on the new node
	// get the pods whose prefix name is docker-registry
	for _, msg := range msgList {
		nodeName := string(msg.Key)
		checkPointFileName := string(msg.Value)

		podIP := pod.Status.PodIP
		rsp, err := http.Get("http://" + podIP + ":5000/v2/_catalog")
		if err != nil {
			log.Error(err, "unable to get the response")
			return err
		}
		body, err := io.ReadAll(rsp.Body)

		if err != nil {
			log.Error(err, "unable to read the body")
			return err
		}

		// check if the body in rsp contains checkPointFileName
		if strings.Contains(string(body), checkPointFileName) {
			// create the pod with the image name, checkPointFileName
			var filteredResult string
			for i, character := range checkPointFileName {
				// Check if the character is a digit
				if unicode.IsDigit(character) {
					// If the character is a digit, break the loop
					filteredResult = checkPointFileName[:i]
					break
				}
			}
			// this is the checkpointFileName, var/lib/kubelet/checkpoints/checkpoint-counter-app-76f6c8d44f-tlt7m_default-counter-2024-05-05T08:19:38Z.tar
			// I want to replace .tar with :latest
			ImageName := strings.ReplaceAll(filteredResult, ".tar", ":latest")
			// also specify which node should this pod deployed on
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: checkPointFileName,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  filteredResult,
							Image: ImageName,
						},
					},
					NodeName: nodeName,
				},
			}
			// create the pod
			clientset, err := util.CreateClientSet()
			if err != nil {
				err := fmt.Errorf("unable to create clientset")
				return err
			}
			_, err = clientset.CoreV1().Pods(nodeName).Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				err := fmt.Errorf("unable to checkpointed pod")
				return err
			}
			break
		}
	}
	wg.Done()
	return nil
}
