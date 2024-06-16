package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ConsumeMessage consumes messages from the Kafka topic for the specified nodeName.
func ConsumeMessage() (kafka.Message, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("ready to fetch message")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
			return kafka.Message{}, nil
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return kafka.Message{}, fmt.Errorf("Failed to fetch message: %v", err)
				}
				continue
			}
			reader.CommitMessages(ctx, msg)

			//the mesage sent by the original sender is key: podName, value: nameSpace/nodeName
			log.Info("msg.Key ", string(msg.Key), " msg.Value ", string(msg.Value))

			// seperate the string with left half string is namespace and right half string is nodeName
			return msg, nil
		}
	}
}
func CommitMessages(msg kafka.Message) error{
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	topic := "delete-pod"
	groupID := "my-group"

	// Create a Kafka consumer (reader)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{bootstrapServers},
		Topic:   topic,
		GroupID: groupID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := reader.CommitMessages(ctx,msg); err != nil {
		log.Error("Failed to commit message: %v", err)
		return err
	}
	return nil
}
