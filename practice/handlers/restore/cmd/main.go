package main

import (
	"restore-daemon/handler"
	"restore-daemon/kafka"
	"strings"

	log "github.com/sirupsen/logrus"
)

func main() {
	// get the message from kafka broker with infinite loop
	log.Info("Start to consume the message from kafka")
	for {
		newPodName, nameSpace, nodeName, err := kafka.ConsumeMessage()
		if err != nil {
			log.Error("unable to get the message from kafka")
		}
		if newPodName == "" || nameSpace == "" || nodeName == "" {
			continue
		}

		log.Info("The ", newPodName, " pod is alive on the ", nodeName, " node")
		log.Info("newpodname: ", newPodName, " namespace: ", nameSpace, " nodeName: ", nodeName)

		// call the delete old pod function, oldPodName is the remain string without the checkpoint-
		oldPodName := newPodName[strings.Index(newPodName, "-")+1:]
		// namespace is wrong, this is the original namespace
		clear := handler.DeleteOldPod(nameSpace, oldPodName)
		if !clear {
			log.Error("unable to delete the old pod")
		}

	}
}
