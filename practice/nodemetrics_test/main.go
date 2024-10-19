package main 
import (
	"context"
	"fmt"
	"os"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	logrus "github.com/sirupsen/logrus"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

)
func createClientSet() (*kubernetes.Clientset, error) {
		// get the kubernetes config
		config, err := rest.InClusterConfig()
	
		if err != nil {
			// If running outside the cluster, use kubeconfig file
			fmt.Println("unable to create clientset by service account")
			kubeconfig := os.Getenv("HOME") + "/.kube/config"
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				panic(err.Error())
			}
		}
	
		// Create Kubernetes client
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Error("unable to create clientset")
			return nil, err
		}
		return clientset, nil
	
}

func main() {
	clientset, err := createClientSet()
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		logrus.Fatalf("Failed to create kubeconfig: %v", err)
	}
	mc, err := metricsclientset.NewForConfig(kubeconfig)
	if err != nil {
		logrus.Fatalf("Failed to create metrics client: %v", err)
	}
	
	nodeName := "workernode01"
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Fatalf("Failed to retrieve node %s: %v", nodeName, err)
	}
	
	// get the cpu and memory usage of the node
	cpuCapacity := node.Status.Allocatable.Cpu().MilliValue()
	memoryCapacity := node.Status.Allocatable.Memory().Value()
	
	nodeMetrics, err := mc.MetricsV1beta1().NodeMetricses().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Fatalf("Failed to get metrics for node %s: %v", nodeName, err)
	}
	cpuUsage := nodeMetrics.Usage["cpu"]
	memoryUsage := nodeMetrics.Usage["memory"]

	// calculate the remaining cpu and memory
	remainingCPU := cpuCapacity - cpuUsage.MilliValue()
	remainingMemory := (memoryCapacity - memoryUsage.Value())/1024/1024
	logrus.Infof("Remaining CPU: %d, Remaining Memory: %d", remainingCPU, remainingMemory)
	cpuLimit := int64(float64(remainingCPU) * 0.2)
	memoryLimit := int64(float64(remainingMemory) * 0.2)
	logrus.Infof("Cpu limit: %d, Memory limit: %d", cpuLimit, memoryLimit)
	// create a pod with the remaining resources
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewMilliQuantity(30, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(5*1024*1024, resource.BinarySI),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewMilliQuantity(3055, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity( 2063459942, resource.BinarySI),
						},
					},
				},
			},
		},
	}
	_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatalf("Failed to create pod: %v", err)
	}


}
