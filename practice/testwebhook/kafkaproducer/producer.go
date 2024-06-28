package kafkaproducer

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ProduceMessage(key string, value string) error {
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	topic := "my-topic"

	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Async:    false,
	}

	// Prepare the message,key and value are read from environment variables
	message := kafka.Message{
		Key:   []byte(key), // Optional: specify a key for the message
		Value: []byte(value),
	}

	ctx := context.Background()
	// Send the message
	err := writer.WriteMessages(ctx, message)
	if err != nil {
		log.Log.Error(err, "Failed to send message")
		return err
	}
	fmt.Println("message sent", key, value)

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Log.Error(err, "Failed to close writer")
	}
	return nil
}
