package main

import (
    "context"
    "fmt"
    "log"
    "github.com/segmentio/kafka-go"
)

func main() {
    // Specify the bootstrap servers and the topic
    bootstrapServers := "my-cluster-kafka-bootstrap:9092"
    topic := "my-topic"

    // Create a Kafka producer
    writer := kafka.Writer{
        Addr:         kafka.TCP(bootstrapServers),
        Topic:        topic,
        Balancer:     &kafka.LeastBytes{},
    }

    // Prepare the message
    message := kafka.Message{
        Key:   []byte("this-key"), // Optional: specify a key for the message
        Value: []byte("Hello, Tony!"),
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

