package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"tony123.tw/util"
)

// integration tests for checkpointing the pod and producing the message
// now before running this test, I should run port-forward the service, svc/my-cluster-kafka-bootstrap
// kubectl port-forward svc/my-cluster-kafka-bootstrap 9092:9092
func TestProduceMessage(t *testing.T) {
	client := GetKubeletClient()
	address := fmt.Sprintf("https://192.168.56.3:10250/checkpoint/default/counter-app-76f6c8d44f-tlt7m/counter")
	rsp, err := CheckpointPod(client, address)
	if err != nil {
		t.Error("unable to checkpoint the pod")
	}
	defer rsp.Body.Close()
	body, err := io.ReadAll(rsp.Body)
	kubeletResponse := &KubeletCheckpointResponse{}
	err = json.Unmarshal(body, kubeletResponse)
	if err != nil {
		log.Log.Error(err, "Error unmarshalling kubelet response")
	}
	producerPod := produceMessage("kafka", "tonyliu666/kafka:latest", "kubenode02", kubeletResponse.Items[0])
	// run the pod
	clientset, err := util.CreateClientSet()
	if err != nil {
		t.Error("unable to create clientset")
	}
	_, err = clientset.CoreV1().Pods("kafka").Create(context.TODO(), producerPod, metav1.CreateOptions{})

	// examine whether the message has been produced
	consumerPod := consumeMessage("kafka", "tonyliu666/kafka-client:latest", "kubenode02", kubeletResponse.Items[0])

	// run the pod
	_, err = clientset.CoreV1().Pods("kafka").Create(context.TODO(), consumerPod, metav1.CreateOptions{})

	if err != nil {
		t.Error("unable to create the consumer pod")
	}

	// check the stdout of the pod existing
	// use like kubectl log kafka-client -n kafka
	// Define the log options
	logOptions := &corev1.PodLogOptions{
		Container: "kafka-client",
		Follow:    true,
	}

	// Fetch the logs for the specified pod and namespace
	req := clientset.CoreV1().Pods("kafka").GetLogs(consumerPod.Name, logOptions)
	stream, err := req.Stream(context.Background())
	if err != nil {
		panic(err.Error())
	}
	defer stream.Close()

	// examine the logs existed
	buf := make([]byte, 1000)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			t.Error("unable to read the logs")
		}
		// successfully run this test
		if n > 0 {
			break
		}
	}
}
func consumeMessage(namespace string, imageName string, key string, value string) *corev1.Pod {
	// set the message whose key is "kubenode02" and value is kubeletResponse.Items[0] as an environment variable
	// consume the message from the kafka broker
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-client",
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kafka-client",
					Image: imageName,
					Env: []corev1.EnvVar{
						{
							Name:  "kafka-key",
							Value: key,
						},
						{
							Name:  "kafka-value",
							Value: value,
						},
					},
				},
			},
		},
	}
	return pod
}

func produceMessage(nameSpace string, imageName string, key string, value string) *corev1.Pod {
	//set the message whose key is "kubenode02" and value is kubeletResponse.Items[0] as an environment variable
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka",
			Namespace: nameSpace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kafka",
					Image: imageName,
					Env: []corev1.EnvVar{
						{
							Name:  "kafka-key",
							Value: key,
						},
						{
							Name:  "kafka-value",
							Value: value,
						},
					},
				},
			},
		},
	}
	return pod
}
