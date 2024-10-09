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
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NodeMonitorReconciler reconciles a NodeMonitor object
type NodeMonitorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	MetricsClient *metricsclientset.Clientset
}

var kubeconfig *rest.Config
var kubeClient *clientset.Clientset
var cpu_usage_rate_maps = make(map[string]float64)
var migrationNode string
var logger logr.Logger

func init() {
	var err error
	// kubeconfig, err = clientcmd.BuildConfigFromFlags("", "/home/tony/.kube/config")
	// if err != nil {
	// 	panic(err)
	// }
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
		logger.Info("kubeconfig", "kubeconfig is", kubeconfig)
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

	// Fetch node metrics
	cpuPercentage, err := r.getCpuUsageForNode(&node, ctx)
	if err != nil {
		logger.Error(err, fmt.Sprint("Failed to get CPU usage for %s node", node.Name))
		return ctrl.Result{}, err
	}

	// update the cpu_usage_rate_maps
	cpu_usage_rate_maps[node.Name] = cpuPercentage

	// If CPU usage exceeds 70%, deploy the custom resource
	if cpuPercentage > 5 {
		logger.Info(fmt.Sprintf("Node %s CPU usage exceeds threshold (%.2f%%). Deploying Migration custom resource!", node.Name, cpuPercentage))

		// lists all the pods on this node
		pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + node.Name,
		})
		if err != nil {
			logger.Error(err, "Failed to list pods")
			return ctrl.Result{}, err
		}
		// find the least cpu usage node to migrate the pod
		for nodeName, cpuUsage := range cpu_usage_rate_maps {
			if cpuUsage < cpuPercentage {
				cpuPercentage = cpuUsage
				migrationNode = nodeName
			}
		}

		// go through each pod in pods and replace the podname field in the yaml with the pod name
		for _, pod := range pods.Items {
			migrationYAML := fmt.Sprintf(`
			apiVersion: api.my.domain/v1alpha1
			kind: Migration
			metadata:
			name: migration-sample
			labels:
				example-webhook-enabled: "true"
			namespace: practice-system
			spec:
			podname: %s
			deployment: 
			namespace: default
			destinationNode: %s
			destinationNamespace: migration
			specify:
			`, pod.Name, migrationNode)
			// Decode the YAML into an unstructured object
			obj := &unstructured.Unstructured{}
			dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(migrationYAML)), 1000)
			if err := dec.Decode(obj); err != nil {
				logger.Error(err, "Failed to decode Migration resource YAML")
				return ctrl.Result{}, err
			}

			// Set the GroupVersionKind for the custom resource
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "api.my.domain",
				Version: "v1alpha1",
				Kind:    "Migration",
			})

			// Apply the custom resource using the controller-runtime client
			err = r.Client.Create(ctx, obj)
			if err != nil {
				logger.Error(err, "Failed to create Migration custom resource")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully created Migration custom resource")
		}
	}

	// Reconcile periodically (adjust timing as needed)
	return ctrl.Result{RequeueAfter: time.Minute}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&corev1.Node{}).
		Complete(r)
}
func (r *NodeMonitorReconciler) getCpuUsageForNode(node *corev1.Node, ctx context.Context) (float64, error) {
	nodeMetrics, err := r.MetricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Failed to get node metrics")
		return -1, err
	}

	logger.Info(fmt.Sprintf("Node %s timestamp: %s", node.Name, nodeMetrics.Timestamp))

	// Calculate CPU usage percentage
	usageCPU := nodeMetrics.Usage.Cpu().MilliValue()
	allocatableCPU := node.Status.Allocatable.Cpu().MilliValue()
	cpuPercentage := float64(usageCPU) / float64(allocatableCPU) * 100
	logger.Info(fmt.Sprintf("Node %s CPU usage: %.2f%%", node.Name, cpuPercentage))
	return cpuPercentage, nil

}
