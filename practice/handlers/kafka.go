package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	util "tony123.tw/util"
)

func ConsumeMessage(nodeName string) ([]kafka.Message, error) {
	// get the message from kafka broker
	// hard-coded only process ten messages at a time
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	// bootstrapServers := "192.168.56.3:32195"
	topic := "my-topic"
	groupID := "my-group"

	// Create a Kafka consumer (reader)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{bootstrapServers},
		Topic:   topic,
		GroupID: groupID,
	})

	defer reader.Close()
	// Consume messages from the topic
	now := time.Now()
	messageList := []kafka.Message{}
	for {
		msg, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatalf("Failed to fetch message: %v", err)
		}
		if string(msg.Key) == nodeName {
			// Commit the offset to acknowledge the message has been processed
			if err := reader.CommitMessages(context.Background(), msg); err != nil {
				log.Fatalf("Failed to commit message: %v", err)
			}
			messageList = append(messageList, msg)
		}
		if time.Since(now) > 3*time.Second {
			return messageList, nil
		}
	}

}
func ProduceMessage(key string, value string) error {
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	// bootstrapServers := "192.168.56.3:32195"
	topic := "my-topic"
	value = util.ModifyCheckpointToImageName(value)

	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// Prepare the message,key and value are read from environment variables
	message := kafka.Message{
		Key:   []byte(key), // Optional: specify a key for the message
		Value: []byte(value),
	}

	// Send the message
	err := writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}
	fmt.Println("message sent")

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Fatalf("Failed to close producer: %v", err)
	}
	return nil
}
