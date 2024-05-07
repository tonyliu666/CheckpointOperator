package main

import (
	"context"
	"fmt"
	"log"
	//"os"

	"github.com/segmentio/kafka-go"
)

func main() {
    // Specify the bootstrap servers, topic, and consumer group ID
    // bootstrapServers := "my-cluster-kafka-bootstrap:9092"
    // bootstrapServers := "my-cluster-kafka-bootstrap:9092"
    bootstrapServers := "192.168.56.3:32195"
    
    topic := "my-topic"
    groupID := "my-group"

    // kafkaKey := os.Getenv("kafka-key")
    // kafkaValue := os.Getenv("kafka-value")
    kafkaKey := "key1"
    kafkaValue := "value1"

    // Create a Kafka consumer (reader)
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:   []string{bootstrapServers},
        Topic:     topic,
        GroupID:   groupID,
        // Partition: 0, // Optional: specify a partition if needed
    })

    defer reader.Close()

    // Consume messages from the topic
    for {
        msg, err := reader.FetchMessage(context.Background())
        if err != nil {
            log.Fatalf("Failed to fetch message: %v", err)
        }
        if string(msg.Key) == kafkaKey && string(msg.Value) == kafkaValue {
            // Commit the offset to acknowledge the message has been processed
            if err := reader.CommitMessages(context.Background(), msg); err != nil {
                log.Fatalf("Failed to commit message: %v", err)
            }
            fmt.Printf("Message: %v\n", string(msg.Value))
        }
    }
}