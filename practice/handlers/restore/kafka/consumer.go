package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ConsumeMessage consumes messages from the Kafka topic for the specified nodeName.
func ConsumeMessage() (string,string,string, error) {
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	topic := "delete-pod"
	groupID := "my-group"

	// Create a Kafka consumer (reader)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{bootstrapServers},
		Topic:   topic,
		GroupID: groupID,
	})

	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
			return "","","",ctx.Err()
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return "","","",nil
				}
				continue
			}
			reader.CommitMessages(ctx, msg)

			//the mesage sent by the original sender is key: podName, value: nameSpace/nodeName
			log.Info("msg.Key ", string(msg.Key), " msg.Value ", string(msg.Value))

			// seperate the string with left half string is namespace and right half string is nodeName
			newPodName := string(msg.Key)
			nameSpace := string(msg.Value)[0:strings.Index(string(msg.Value), "/")]
			nodeName := string(msg.Value)[strings.Index(string(msg.Value), "/")+1:]
			return newPodName, nameSpace, nodeName, nil
		}
	}
}
