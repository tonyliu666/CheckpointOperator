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
	DestinationNamespace string
	Specify              []string
)

func FillinGlobalVariables(migration *apiv1alpha1.Migration) {
	PodName = migration.Spec.PodName
	Deployment = migration.Spec.Deployment
	SourceNamespace = migration.Spec.Namespace
	DestinationNode = migration.Spec.DestinationNode
	DestinationNamespace = migration.Spec.DestinationNamespace
	Specify = migration.Spec.Specify
}
