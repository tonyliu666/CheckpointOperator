package handlers

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/segmentio/kafka-go"
	util "tony123.tw/util"
)

// ConsumeMessage consumes messages from the Kafka topic for the specified nodeName.
func ConsumeMessage(nodeName string) ([]kafka.Message, error) {
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
	messageList := []kafka.Message{}
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
			return messageList, ctx.Err()
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return messageList, nil
				}
				log.Log.Error(err, "Failed to fetch message")
				continue
			}

			// fmt.Println("msg.Key: ", string(msg.Key), "msg.Value:", string(msg.Value), "nodeName: ", nodeName)
			log.Log.Info("Received message",
				"msg.Key", string(msg.Key),
				"msg.Value", string(msg.Value),
				"nodeName", nodeName,
			)
			if string(msg.Key) == "" || string(msg.Value) == "" {
				// commit the message
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Log.Error(err, "Failed to commit message")
				}
				continue
			}

			if string(msg.Key) == nodeName {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					fmt.Printf("Failed to commit message: %v", err)
					return messageList, err
				}
				messageList = append(messageList, msg)
			}
		}
	}
}
func ConsumeMessageFromDifferentTopics(nodeName string) ([]kafka.Message, error) {
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
	messageList := []kafka.Message{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Log.Info("ready to fetch message", "topic", topic)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
			return messageList, ctx.Err()
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return messageList, nil
				}
				continue
			}

			// fmt.Println("msg.Key: ", string(msg.Key), "msg.Value:", string(msg.Value), "nodeName: ", nodeName)
			log.Log.Info("msg.Key", string(msg.Key), "msg.Value", string(msg.Value), "topic", topic)

			if err := reader.CommitMessages(ctx, msg); err != nil {
				fmt.Printf("Failed to commit message: %v", err)
				return messageList, err
			}
			messageList = append(messageList, msg)
			log.Log.Info("messageList", messageList)
		}
	}
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
		Async:    false,
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
	// fmt.Println("message sent", key, value)
	log.Log.Info("message sent", " key", key, " value", value)

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Log.Error(err, "Failed to close writer")
	}
	return nil
}

func ProduceMessageToDifferentTopics(key string, nameSpace string, nodeName string) error {
	bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	topic := "delete-pod"

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
		Value: []byte(nameSpace + "/" + nodeName),
	}

	// Send the message
	err := writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}
	log.Log.Info("message sent", "key", key, "value", nameSpace+"/"+nodeName)

	// Close the producer
	if err := writer.Close(); err != nil {
		log.Log.Error(err, "Failed to close writer")
	}
	return nil
}
