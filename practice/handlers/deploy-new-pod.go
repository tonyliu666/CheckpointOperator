package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	util "tony123.tw/util"
)

func DeployPodOnNewNode(pod *corev1.Pod, wg *sync.WaitGroup) error {
	// get the message from kafka broker
	// TODO: Consume the message from the Kafka broker with the real go codes
	// probablt, this function will miss some messages, because it only process ten messages at a time
	msgList, err := ConsumeMessage(pod.Spec.NodeName)
	if err != nil {
		return err
	}

	// TODO: Deploy a new pod on the new node
	// get the pods whose prefix name is docker-registry
	for _, msg := range msgList {
		nodeName := string(msg.Key)
		podName := string(msg.Value)
		podIP := pod.Status.PodIP
		// can't be accessed from outside the cluster
		registryAPIAddr := fmt.Sprintf("http://%s:5000/v2/_catalog", podIP)
		fmt.Println("registryAPIAddr: ", registryAPIAddr)
		rsp, err := http.Get(registryAPIAddr)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		fmt.Println("checkPointPodName: ", podName, "body", string(body))
		// check if the body in rsp contains checkPointFileName
		if strings.Contains(string(body), podName) {
			// I want to replace .tar with :latest, maybe the tag is not latest, so I want to replace the tag with latest
			
			ImageName := podName + ":latest"
			ImageRegistry := podIP + ":5000"+"/"+ImageName
			fmt.Println("ImageRegistry ", ImageRegistry)
			// pull image from the docker registry pod
			// also specify which node should this pod deployed on
			// create a buildah pull pod
			err = BuildahPodPullImage(pod, ImageRegistry)
			if err != nil {
				fmt.Println("error: ", err)
				err := fmt.Errorf("unable to pull image from the docker registry pod")
				return err
			}
			// 
			migratePod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podName,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podName,
							Image: ImageRegistry,
						},
					},
					NodeName: nodeName,
				},
			}
			
			// create the pod
			clientset, err := util.CreateClientSet()
			if err != nil {
				fmt.Println("error: ", err)
				err := fmt.Errorf("unable to create clientset")
				return err
			}
			_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), migratePod, metav1.CreateOptions{})
			if err != nil {
				fmt.Println("error: ", err)
				err := fmt.Errorf("unable to checkpointed pod")
				return err
			}
			break
		}
	}
	wg.Done()
	return nil
}
