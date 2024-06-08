package main

import (
	"fmt"
	"restore-daemon/handler"
	"restore-daemon/kafka"
	"strings"

	log "github.com/sirupsen/logrus"
)

func main() {
	// get the message from kafka broker with infinite loop
	for {
		newPodName, nameSpace, nodeName, err := kafka.ConsumeMessage()
		if err != nil {
			log.Error("unable to get the message from kafka")
		}
		log.Info("newPodName ", newPodName, " nameSpace ", nameSpace, " nodeName ", nodeName)

		if newPodName == "" || nameSpace == "" || nodeName == "" {
			continue
		}
		alive, state := handler.AliveCheck(nameSpace, newPodName, nodeName)

		if alive {
			log.Info("The ", newPodName, " pod is alive on the", nodeName, " node")

			// call the delete old pod function, oldPodName is the remain string without the checkpoint-
			oldPodName := newPodName[strings.Index(newPodName, "-")+1:]
			clear:=handler.DeleteOldPod(nameSpace, oldPodName)
			if !clear{
				log.Error("unable to delete the old pod")
			}
		} else {
			// TODO: handle the case that the pod is not alive
			fmt.Println(state)
			log.Error("The", newPodName, " pod is not alive on the", nodeName, " node")
		}
	}
}
