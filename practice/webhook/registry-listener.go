// create a webhook with the package, github.com/adnanh/webhook
// and register a listener to handle the incoming requests from docker registry
package main

import (
	wehook "github.com/adnanh/webhook/webhook"
	"k8s.io/kube-openapi/cmd/openapi-gen/args"
)

func main() {
	// Define a slice of hooks
	hooks := []webhook.Hook{
		{
			Name:   "registry-listener",
			Events: []string{"push"},
			Match: webhook.Match{
				Type:   "value",
				Header: "User-Agent",
				Value:  "Docker",
			},
			// send the message with key,value with docker-registry and image name to kafka broker
			Execute: []webhook.Command{
				{
					Command: "go run kafkaproducer/message.go",
					args:    []string{"-key", "docker-registry", "-value", "{{.repository}}"},
			},
		},
	}

}
