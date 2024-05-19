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
		fmt.Println("msg.Key: ", string(msg.Key),"msg.Value:", string(msg.Value), "nodeName: ", nodeName)
		if err != nil {
			log.Fatalf("Failed to fetch message: %v", err)
		}
		if string(msg.Key) == nodeName {
			// Commit the offset to acknowledge the message has been processed
			if err := reader.CommitMessages(context.Background(), msg); err != nil {
				log.Fatalf("Failed to commit message: %v", err)
			}
			messageList = append(messageList, msg)
		}else{
			break 
		}
		if time.Since(now) > 1*time.Second {
			break
		}
	}
	return messageList, nil

}
func ProduceMessage(key string, value string) error {
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	topic := "my-topic"
	value = util.ModifyCheckpointToImageName(value)

	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Async: true,
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
	fmt.Println("message sent", key, value)

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Fatalf("Failed to close producer: %v", err)
	}
	return nil
}
