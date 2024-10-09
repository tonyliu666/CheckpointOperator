package handlers

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDecodeCustomResource(t *testing.T) {
	// Decode the YAML into an unstructured object
	obj := &unstructured.Unstructured{}
	err := DecodeCustomResource(obj, "test-pod", "test-node")
	if err != nil {
		t.Errorf("DecodeCustomResource failed: %v", err)
	}
}
