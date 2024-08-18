package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testwebhook/k8sclient"
	"testwebhook/kafkaproducer"
	"testwebhook/util"
	"time"

	v1 "k8s.io/api/core/v1"
	// "sigs.k8s.io/controller-runtime/pkg/log"
	log "github.com/sirupsen/logrus"
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

var registryPodList *v1.PodList


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
			log.Info("Event received ", "event", e)
			// if the mediaType is equal to application/vnd.oci.image.manifest.v1+json, then mean all the image content has been pushed
			if e.Target.MediaType == "application/vnd.oci.image.manifest.v1+json" || e.Target.MediaType == "application/vnd.docker.distribution.manifest.v2+json" {
				// hostIP is the data like this 10.244.16.247:5000, I want to get 10.244.16.247
				registryPodIP := e.Request.Host
				log.Info("registryPodIP first:" , registryPodIP)
				registryPodIP = strings.Split(registryPodIP, ":")[0]
				log.Info("registryPodIP second:" , registryPodIP)
				// get the node name of the pod
				nodeName, err := util.GetNodeNameByHostIP(registryPodIP, registryPodList)
				// send messages to kafka broker
				if err != nil {
					log.Error(err, "unable to get node name by host IP")
				}
				if nodeName == "" {
					log.Warn("Node name not found for registryPodIP: ", registryPodIP)
					continue 
				}
				err = kafkaproducer.ProduceMessage(nodeName, e.Target.Repository)
				if err != nil {
					log.Error(err, "unable to produce message to kafka broker")
				}
			}
		}
	}
	// Respond with HTTP 200 OK status
	w.WriteHeader(http.StatusOK)
}

func main() {
	var err error
	registryPodList, err = k8sclient.ListPods("docker-registry", "app=docker-registry")
	if err != nil {
		log.Error(err, "unable to list pods")
	}

	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
