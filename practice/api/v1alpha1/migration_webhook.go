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

package v1alpha1

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"tony123.tw/util"
)

// log is for logging in this package.
var migrationlog = logf.Log.WithName("migration-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Migration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-api-my-domain-v1alpha1-migration,mutating=true,failurePolicy=fail,sideEffects=None,groups=api.my.domain,resources=migrations,verbs=create;update,versions=v1alpha1,name=mmigration.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Migration{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Migration) Default() {
	migrationlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-api-my-domain-v1alpha1-migration,mutating=false,failurePolicy=fail,sideEffects=None,groups=api.my.domain,resources=migrations,verbs=create;update,versions=v1alpha1,name=vmigration.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Migration{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Migration) ValidateCreate() (admission.Warnings, error) {
	migrationlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.

	return nil, r.validateMigration()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Migration) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	migrationlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, r.validateMigration()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Migration) ValidateDelete() (admission.Warnings, error) {
	migrationlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
func (r *Migration) validateMigration() error {
	// r.Spec.PodName, r.Spec.deployment and r.Spec.specify only one of them can be set
	if r.Spec.PodName != "" && r.Spec.Deployment != "" {
		return fmt.Errorf("only one of podname and deployment can be set")
	}
	if r.Spec.PodName != "" && len(r.Spec.Specify) > 0 {
		return fmt.Errorf("only one of podname and specify can be set")
	}
	if r.Spec.Deployment != "" && len(r.Spec.Specify) > 0 {
		return fmt.Errorf("only one of deployment and specify can be set")
	}

	// r.Spec.DestinationNode, r.Spec.DestinationNamespace and r.Spec.Namespace must be set
	if r.Spec.DestinationNode == "" {
		return fmt.Errorf("destinationNode must be set")
	}
	if r.Spec.DestinationNamespace == "" {
		return fmt.Errorf("destinationNamespace must be set")
	}
	if r.Spec.Namespace == "" {
		return fmt.Errorf("namespace must be set")
	}

	// check whether the pod with the PodName exists in the namespace
	// create a clientset
	clientset, err := createClientSet()
	if r.Spec.PodName != "" {
		if err != nil {
			return fmt.Errorf("unable to create clientset %v", err)
		}
		_, err = clientset.CoreV1().Pods(r.Spec.Namespace).Get(context.TODO(), r.Spec.PodName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("your specified pod %s does not exist in namespace %s", r.Spec.PodName, r.Spec.Namespace)
		}
	}
	if r.Spec.Deployment != "" {
		// check whether the deployment with the Deployment name exists in the namespace
		_, err = clientset.AppsV1().Deployments(r.Spec.Namespace).Get(context.TODO(), r.Spec.Deployment, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("your specified deployment %s does not exist in namespace %s", r.Spec.Deployment, r.Spec.Namespace)
		}
	}
	if len(r.Spec.Specify) > 0 {
		// check whether the specified pods exist in the namespace
		for _, podName := range r.Spec.Specify {
			_, err = clientset.CoreV1().Pods(r.Spec.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("your specified pod %s does not exist in namespace %s", podName, r.Spec.Namespace)
			}
		}
	}
	// check the custom resource has been deployed during the process
	// if r.Spec.PodName + r.Spec.Namespace  in util.ProcessPodsMap then return error
	if r.Spec.PodName != "" {
		if _, ok := util.ProcessPodsMap[r.Spec.PodName]; ok {
			return fmt.Errorf("the pod %s in namespace %s is being processed sorry for this moment only one specific pod name among the cluster can be accepted", r.Spec.PodName, r.Spec.Namespace)
		}
	}
	if r.Spec.Specify != nil {
		for _, podName := range r.Spec.Specify {
			if _, ok := util.ProcessPodsMap[podName]; ok {
				return fmt.Errorf("the pod %s  is being processed. sorry for this moment only one specific pod name among the cluster can be accepted", podName)
			}
		}
	}

	return nil
}
func createClientSet() (*kubernetes.Clientset, error) {
	// get the kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If running outside the cluster, use kubeconfig file
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create clientset %v", err)
	}
	return clientset, nil
}
