package handlers

import (
	"bytes"
	"fmt"

	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func DecodeCustomResource(obj *unstructured.Unstructured, podName string, destinationNode string) error {
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
`, podName, destinationNode)
	// Decode the YAML into an unstructured object

	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(migrationYAML)), 1000)
	if err := dec.Decode(obj); err != nil {
		log.Println("error decoding yaml")
		return err
	}

	// Set the GroupVersionKind for the custom resource
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "api.my.domain",
		Version: "v1alpha1",
		Kind:    "Migration",
	})

	return nil

}
