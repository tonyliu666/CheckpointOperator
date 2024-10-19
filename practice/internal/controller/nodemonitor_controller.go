/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"tony123.tw/handlers"
)

// NodeMonitorReconciler reconciles a NodeMonitor object
type NodeMonitorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	MetricsClient *metricsclientset.Clientset
}

type PodMemoryUsage struct {
	pod         corev1.Pod
	memoryUsage int64
}

type PodCPUUsage struct {
	pod      corev1.Pod
	cpuUsage int64
}

var kubeconfig *rest.Config
var kubeClient *clientset.Clientset
var cpu_usage_rate_maps = make(map[string]float64)
var memory_usage_rate_maps = make(map[string]float64)
var migrationCPUNode string
var migrationMemoryNode string
var logger logr.Logger

func init() {
	var err error
	kubeconfig, err = rest.InClusterConfig()
	if err != nil {
		log.Log.Error(err, "Failed to create kubeconfig")
	}

	kubeClient, err = clientset.NewForConfig(kubeconfig)
	if err != nil {
		log.Log.Error(err, "Failed to create kubeClient")
	}
}

// +kubebuilder:rbac:groups=api.my.domain,resources=nodemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.my.domain,resources=nodemonitors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.my.domain,resources=nodemonitors/finalizers,verbs=update
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeMonitor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *NodeMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger = log.FromContext(ctx)
	if r.MetricsClient == nil {
		mc, err := metricsclientset.NewForConfig(kubeconfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		r.MetricsClient = mc
	}

	// Fetch the Node instance
	var node corev1.Node
	err := r.Get(ctx, req.NamespacedName, &node)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Node not found")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Node")
		return ctrl.Result{}, err
	}
	// if node labels exists node-role.kubernetes.io/control-plane, return
	if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
		return ctrl.Result{}, nil
	}

	// Fetch node metrics
	cpuPercentage, err := r.getCpuUsageOnNode(&node, ctx)
	if err != nil {
		logger.Error(err, "Failed to get CPU usage on node")
		return ctrl.Result{}, err
	}

	// update the cpu_usage_rate_maps
	cpu_usage_rate_maps[node.Name] = cpuPercentage

	// If CPU usage exceeds 70%, deploy the custom resource
	if cpuPercentage > 60 {
		logger.Info(fmt.Sprintf("Node %s CPU usage exceeds threshold (%.2f%%). Deploying Migration custom resource!", node.Name, cpuPercentage))

		TopFivePodsByCPU, err := r.getTopFivePodsByCPU(ctx, node.Name)

		if err != nil {
			logger.Error(err, "Failed to list pods")
			return ctrl.Result{}, err
		}
		// find the least cpu usage node to migrate the pod
		for nodeName, cpuUsage := range cpu_usage_rate_maps {
			if cpuUsage < cpuPercentage {
				cpuPercentage = cpuUsage
				migrationCPUNode = nodeName
			}
		}

		//logger.Info("cpu usage rate maps", "cpu_usage_rate_maps", cpu_usage_rate_maps)

		// go through each pod in pods and replace the podname field in the yaml with the pod name
		for _, item := range TopFivePodsByCPU {
			// avoid migrating kube-system pods
			ok := checkIsValidNamespace(item.pod.Namespace)
			if !ok {
				continue
			}
			logger.Info(fmt.Sprintf("pod %s is going to be migrated to node %s ", item.pod.Name, migrationCPUNode))
			logger.Info(fmt.Sprintf("Pod %s CPU usage: %d millicores", item.pod.Name, item.cpuUsage))

			err = handlers.DoSSA(ctx, kubeconfig, &item.pod, migrationCPUNode)
			if err != nil {
				logger.Error(err, "Failed to do SSA")
			}
		}
	}

	// get memory usage rate
	memoryUsageRate, err := r.getNodeMemoryUsageRate(ctx, node.Name)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to count memory usage rate for %s node", node.Name))
		return ctrl.Result{}, err
	}
	// logger.Info(fmt.Sprintf("Node %s memory usage rate: %.2f%%", node.Name, memoryUsageRate))
	// update the memory_usage_rate_maps
	memory_usage_rate_maps[node.Name] = memoryUsageRate
	// logger.Info("memory usage rate maps", "memory_usage_rate_maps", memory_usage_rate_maps)

	if memoryUsageRate > 70 {
		// find the least memory usage node to migrate the pod
		for nodeName, memoryUsage := range memory_usage_rate_maps {
			if memoryUsage < memoryUsageRate {
				memoryUsageRate = memoryUsage
				migrationMemoryNode = nodeName
			}
		}
		// lists all the pods on this node
		topMemoryUsagesPod, err := r.getTopFivePodsByMemory(ctx, node.Name)
		if err != nil {
			logger.Error(err, "Failed to get top 5 memory usage pods")
			return ctrl.Result{}, err
		}
		for _, item := range topMemoryUsagesPod {
			// avoid migrating kube-system pods
			ok := checkIsValidNamespace(item.pod.Namespace)
			if !ok {
				continue
			}
			logger.Info(fmt.Sprintf("Pod %s memory usage: %d bytes", item.pod.Name, item.memoryUsage))

			err = handlers.DoSSA(ctx, kubeconfig, &item.pod, migrationMemoryNode)
			if err != nil {
				logger.Error(err, "Failed to do SSA")
			}
		}

	}
	// Reconcile periodically (adjust timing as needed)
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&corev1.Node{}).
		Complete(r)
}
func (r *NodeMonitorReconciler) getCpuUsageOnNode(node *corev1.Node, ctx context.Context) (float64, error) {
	nodeMetrics, err := r.MetricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Failed to get node metrics")
		return -1, err
	}

	// Calculate CPU usage percentage
	usageCPU := nodeMetrics.Usage.Cpu().MilliValue()
	allocatableCPU := node.Status.Allocatable.Cpu().MilliValue()
	cpuPercentage := float64(usageCPU) / float64(allocatableCPU) * 100
	// logger.Info(fmt.Sprintf("Node %s CPU usage: %.2f%%", node.Name, cpuPercentage))
	return cpuPercentage, nil
}

func (r *NodeMonitorReconciler) getNodeMemoryUsageRate(ctx context.Context, nodeName string) (float64, error) {
	// Get node metrics (used memory)
	nodeMetrics, err := r.MetricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Node metrics not available
			return -1, fmt.Errorf("node metrics not found for node %s", nodeName)
		}
		return -1, fmt.Errorf("failed to get node metrics: %v", err)
	}

	// Get the used memory from nodeMetrics
	usedMemory := nodeMetrics.Usage[corev1.ResourceMemory]
	usedMemoryBytes := usedMemory.Value() // Memory usage in bytes

	// Get the total memory (capacity) from the Node object
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return -1, fmt.Errorf("failed to get node information: %v", err)
	}

	// Get the total memory from node's capacity
	totalMemory := node.Status.Capacity[corev1.ResourceMemory]
	totalMemoryBytes := totalMemory.Value() // Total memory in bytes

	// Calculate memory usage rate
	memoryUsageRate := (float64(usedMemoryBytes) / float64(totalMemoryBytes)) * 100

	return memoryUsageRate, nil
}

// Get top 5 pods with the most memory usage rate
func (r *NodeMonitorReconciler) getTopFivePodsByMemory(ctx context.Context, nodeName string) ([]PodMemoryUsage, error) {
	// 1. List all the pods on the node
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node: %w", err)
	}

	// 2. Collect memory usage for each pod
	var podMemoryUsages []PodMemoryUsage
	for _, pod := range pods.Items {
		podMetrics, err := r.MetricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Failed to get metrics for pod %s: %v\n", pod.Name, err)
			continue
		}

		// Calculate total memory usage for the pod
		totalMemoryUsage := int64(0)
		for _, container := range podMetrics.Containers {
			if memUsage, ok := container.Usage[corev1.ResourceMemory]; ok {
				totalMemoryUsage += memUsage.Value() // Memory usage in bytes
			}
		}

		podMemoryUsages = append(podMemoryUsages, PodMemoryUsage{
			pod:         pod,
			memoryUsage: totalMemoryUsage,
		})
	}

	// 3. Sort pods by memory usage
	sort.Slice(podMemoryUsages, func(i, j int) bool {
		return podMemoryUsages[i].memoryUsage > podMemoryUsages[j].memoryUsage
	})

	// 4. Return the top 5 pods
	topPods := podMemoryUsages
	if len(podMemoryUsages) > 5 {
		topPods = podMemoryUsages[:5]
	}

	return topPods, nil
}
func (r *NodeMonitorReconciler) getTopFivePodsByCPU(ctx context.Context, nodeName string) ([]PodCPUUsage, error) {
	// 1. List all the pods on the node
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node: %w", err)
	}

	// 2. Collect memory usage for each pod
	var TopFivePodCPUUsage []PodCPUUsage
	for _, pod := range pods.Items {
		podMetrics, err := r.MetricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Failed to get metrics for pod %s: %v\n", pod.Name, err)
			continue
		}

		// Calculate total memory usage for the pod
		totalCPUUsage := int64(0)
		for _, container := range podMetrics.Containers {
			if cpuUsage, ok := container.Usage[corev1.ResourceCPU]; ok {
				totalCPUUsage += cpuUsage.Value() // Memory usage in bytes
			}
		}

		TopFivePodCPUUsage = append(TopFivePodCPUUsage, PodCPUUsage{
			pod:      pod,
			cpuUsage: totalCPUUsage,
		})
	}
	return TopFivePodCPUUsage, nil
}

func checkIsValidNamespace(namespace string) bool {
	switch namespace {
	case "kube-system", "kube-public", "kube-node-lease":
		return false
	case "calico-apiserver", "calico-system", "cert-manager", "tigera-operator":
		return false
	case "kafka", "docker-registry", "practice-system", "restore":
		return false
	}

	return true
}
