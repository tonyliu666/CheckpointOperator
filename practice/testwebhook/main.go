package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testwebhook/k8sclient"
	"testwebhook/kafkaproducer"
	"time"
	"testwebhook/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Event struct {
	Events []struct {
		ID        string    `json:"id"`
		Timestamp time.Time `json:"timestamp"`
		Action    string    `json:"action"`
		Target    struct {
			MediaType  string `json:"mediaType"`
			Size       int    `json:"size"`
			Digest     string `json:"digest"`
			Length     int    `json:"length"`
			Repository string `json:"repository"`
			URL        string `json:"url"`
		} `json:"target"`
		Request struct {
			ID        string `json:"id"`
			Addr      string `json:"addr"`
			Host      string `json:"host"`
			Method    string `json:"method"`
			UserAgent string `json:"userAgent"`
		} `json:"request"`
		Actor struct {
			Name string `json:"name"`
		} `json:"actor"`
		Source struct {
			Addr       string `json:"addr"`
			InstanceID string `json:"instanceID"`
		} `json:"source"`
	} `json:"events"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var event Event
	// Decode the JSON payload from the request body
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		fmt.Printf("Error decoding JSON: %v\n", err)
		return
	}

	for _, e := range event.Events {
		// create an event sent to kafka broker
		// TODO: sent messages to kafka broker(kubeletResponse.Items[0],and what timestamp)
		if e.Action == "push" {
			// Print the event for debugging purposes
			fmt.Println("Event: ", e)
			// if the mediaType is equal to application/vnd.oci.image.manifest.v1+json, then mean all the image content has been pushed
			if e.Target.MediaType == "application/vnd.oci.image.manifest.v1+json" || e.Target.MediaType == "application/vnd.docker.distribution.manifest.v2+json" {
				client, err := k8sclient.CreateClientSet()
				if err != nil {
					log.Log.Error(err, "unable to create clientset")
				}
				// TODO: get the docker registry pod list and the pod deployed on which node
				registryPodList, err := client.CoreV1().Pods("docker-registry").List(context.TODO(), metav1.ListOptions{
					LabelSelector: "app=docker-registry",
				})
				// hostIP is the data like this 10.244.16.247:5000, I want to get 10.244.16.247
				registryPodIP := e.Request.Host
				registryPodIP = strings.Split(registryPodIP, ":")[0]
				fmt.Println("registryPodIP: ", registryPodIP)
				// get the node name of the pod
				nodeName, err := util.GetNodeNameByHostIP(registryPodIP, registryPodList)
				fmt.Println("nodeName: ", nodeName, "repository: ", e.Target.Repository)
				// send messages to kafka broker
				err = kafkaproducer.ProduceMessage(nodeName, e.Target.Repository)
				if err != nil {
					log.Log.Error(err, "unable to produce message to kafka broker")
				}
			}
		}
	}
	// Respond with HTTP 200 OK status
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
