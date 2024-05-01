package main

import (
    "context"
    "fmt"
    "log"
    "github.com/segmentio/kafka-go"
)

func main() {
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
    for {
        msg, err := reader.FetchMessage(context.Background())
        if err != nil {
            log.Fatalf("Failed to fetch message: %v", err)
        }

        // Process the message
        fmt.Printf("Received message: key = %s, value = %s\n", string(msg.Key), string(msg.Value))

        // Commit the offset to acknowledge the message has been processed
        if err := reader.CommitMessages(context.Background(), msg); err != nil {
            log.Fatalf("Failed to commit message: %v", err)
        }
    }

    // Close the consumer
    if err := reader.Close(); err != nil {
        log.Fatalf("Failed to close reader: %v", err)
    }
}
