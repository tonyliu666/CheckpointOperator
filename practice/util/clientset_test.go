package util

import "testing"

func TestCreateClientSet(t *testing.T) {
	// get the kubernetes clientset
	_, err := CreateClientSet()
	if err != nil {
		t.Error("unable to get the clientset")
	}
}
