package util

import (
	apiv1alpha1 "tony123.tw/api/v1alpha1"
)

var (
	// record the variable name in migration custom resource as global variables
	// so that I can use them in different packages
	PodName              string
	Deployment           string
	SourceNamespace      string
	DestinationNode      string
	CheckpointPodName   []string
)
func InitializeCheckpointPodList() {
	CheckpointPodName = make([]string, 0)
}
func FillinGlobalVariables(migration *apiv1alpha1.Migration) {
	PodName = migration.Spec.Podname
	Deployment = migration.Spec.Deployment
	SourceNamespace = migration.Spec.Namespace
	DestinationNode = migration.Spec.Destination
}