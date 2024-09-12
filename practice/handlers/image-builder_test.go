package handlers

import (
	"os"
	"testing"
)

func TestImageBuilder(t *testing.T) {
	// file, err := os.ReadFile("test.tar")
	// file, err := os.ReadFile("nginx.tar")
	file, err := os.ReadFile("nginx.tar")
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile("test2.tar", file, 0644)
	if err != nil {
		t.Error(err)
	}
	TempDir := "."
	err = CreateOCIImage("test2.tar", TempDir, "nginx", "checkpoint")
	if err != nil {
		t.Errorf("Expected tar, received %v", err)
	}
}
