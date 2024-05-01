package handlers

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func ProduceMessage(checkpointFileName string, nodeName string) {
	// Specify the bootstrap servers and the topic
	bootstrapServers := "my-cluster-kafka-bootstrap:9092"
	topic := "my-topic"
	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// Prepare the message
	message := kafka.Message{
		Key:   []byte(nodeName), // Optional: specify a key for the message
		Value: []byte(checkpointFileName),
	}
	// send the message
	err := writer.WriteMessages(context.Background(), message)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	log.Println("Kafka Message sent successfully.")
	// Close the producer
	if err := writer.Close(); err != nil {
		log.Fatalf("Failed to close producer: %v", err)
	}
}

func ConsumeMessage() (msg kafka.Message) {
	// Specify the bootstrap servers, topic, and consumer group ID
	bootstrapServers := "my-cluster-kafka-bootstrap:9092"
	topic := "my-topic"
	groupID := "my-group"

	// Create a Kafka consumer (reader)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{bootstrapServers},
		Topic:     topic,
		GroupID:   groupID,
		Partition: 0, // Optional: specify a partition if needed
	})

	// Consume messages from the topic
	msg, err := reader.FetchMessage(context.Background())
	if err != nil {
		log.Fatalf("Failed to fetch message: %v", err)
	}

	// Process the message
	log.Printf("Received message: key = %s, value = %s\n", string(msg.Key), string(msg.Value))
	// Commit the offset to acknowledge the message has been processed
	if err := reader.CommitMessages(context.Background(), msg); err != nil {
		log.Fatalf("Failed to commit message: %v", err)
	}
	
	// Close the consumer
	if err := reader.Close(); err != nil {
		log.Fatalf("Failed to close reader: %v", err)
	}
	return msg
}
