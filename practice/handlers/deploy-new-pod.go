package handlers

import (
	"context"
	"fmt"

	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
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
	buildahchecker, err := util.CheckImageIDExistOnNode(imageIDList, dstNode)
	if err != nil {
		log.Log.Error(err, "unable to check image id")
		return fmt.Errorf("unable to check image id: %w", err)
	}
	// check the check-image-id job is finished or not
	// set the context for the time limit of the job

	err = util.CheckJobStatus(buildahchecker.Name, "Succeeded")
	if err != nil {
		log.Log.Error(err, "unable to check job status")
		return fmt.Errorf("unable to check job status: %w", err)
	}
	return nil
}
func countRemainingNodeCPUMeory(nodeName string) (int64, int64, error) {
	clientset, err := util.CreateClientSet()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to create clientset: %w", err)
	}
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to retrieve node %s: %w", nodeName, err)
	}
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create kubeconfig: %w", err)
	}
	mc, err := metricsclientset.NewForConfig(kubeconfig)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create metrics client: %w", err)
	}

	// get the cpu and memory usage of the node
	cpuCapacity := node.Status.Allocatable.Cpu().MilliValue()
	memoryCapacity := node.Status.Allocatable.Memory().Value()

	nodeMetrics, err := mc.MetricsV1beta1().NodeMetricses().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to retrieve node metrics %s: %w", nodeName, err)
	}
	cpuUsage := nodeMetrics.Usage["cpu"]
	memoryUsage := nodeMetrics.Usage["memory"]

	// calculate the remaining cpu and memory
	remainingCPU := cpuCapacity - cpuUsage.MilliValue()
	remainingMemory := memoryCapacity - memoryUsage.Value()
	return remainingCPU, remainingMemory, nil

}
func createNewPod(migratePod *corev1.Pod, nameSpace string) (*corev1.Pod, error) {
	clientset, err := util.CreateClientSet()
	if err != nil {
		return nil, fmt.Errorf("unable to create clientset: %w", err)
	}
	_, err = clientset.CoreV1().Pods(nameSpace).Create(context.TODO(), migratePod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to create pod: %w", err)
	}
	return migratePod, nil
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
		log.Log.Info("oldPodName", "oldPodName", oldPodName)

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
		// before deploy the new pod on the destination node, check the remaining cpu and memory
		remainingCPU, remainingMemory, err := countRemainingNodeCPUMeory(nodeName)
		if err != nil {
			log.Log.Error(err, "unable to count the remaining cpu and memory")
			return fmt.Errorf("unable to count the remaining cpu and memory: %w", err)
		}
		// TODO: need to fix this with the resource request and limit
		memoryLimit := int64(float64(remainingMemory) * 0.2)
		cpuLimit := int64(float64(remainingCPU) * 0.2)

		migratePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  podName,
						Image: imageLocation,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(1, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(4*1024*1024, resource.BinarySI),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuLimit, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(memoryLimit, resource.BinarySI),
							},
						},
					},
				},
				NodeName: info.DestinationNode,
			},
		}

		// make sure the destantion namespace is created
		err = checkDestinationNameSpaceExists(info.DestinationNamespace)
		if err != nil {
			log.Log.Error(err, "unable to check destination namespace")
			return fmt.Errorf("unable to check destination namespace: %w", err)
		}
		newpod, err := createNewPod(migratePod, info.DestinationNamespace)
		if err != nil {
			log.Log.Error(err, "unable to create new pod")
			return fmt.Errorf("unable to create new pod: %w", err)
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
