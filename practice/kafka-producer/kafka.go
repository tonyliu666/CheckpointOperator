package main

import (
	"context"
	"fmt"
	"log"
	"os"

	// "os"

	"github.com/segmentio/kafka-go"
)

func main() {
	bootstrapServers := "my-cluster-kafka-bootstrap:9092"
	// bootstrapServers := "192.168.56.5:30092"
	// bootstrapServers := "localhost:9092"
	// bootstrapServers := "minikube-kafka-testing.io:30571"
	topic := "my-topic"

	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	key := os.Getenv("kafka-key")
	value := os.Getenv("kafka-value")

	// Prepare the message,key and value are read from environment variables
	message := kafka.Message{
		Key:   []byte(key), // Optional: specify a key for the message
		Value: []byte(value),
		// Key:  []byte("key1"), // Optional: specify a key for the message
		// Value: []byte("value1"),
	}

	// Send the message
	err := writer.WriteMessages(context.Background(), message)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Message sent successfully.")

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Fatalf("Failed to close producer: %v", err)
	}

}
