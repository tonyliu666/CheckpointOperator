package handlers

import (
	"context"
	"fmt"
	"log"
	"testing"
	"github.com/segmentio/kafka-go"
	"tony123.tw/util"
)

// integration tests for checkpointing the pod and producing the message
// now before running this test, I should run port-forward the service, svc/my-cluster-kafka-bootstrap
// kubectl port-forward svc/my-cluster-kafka-bootstrap 9092:9092
func TestProduceMessage(t *testing.T) {
	checkpointResponse := &KubeletCheckpointResponse{}
	err := checkpointResponse.RandomCheckpointPod("default")
	if err != nil {
		t.Error("Error unmarshalling kubelet response")
	}
	err = produceMessage("kubenode02", checkpointResponse.Items[0])
	if err != nil {
		t.Error("Error producing kafka message")
	}
	// examine whether the message has been produced
	err = consumeMessage("kubenode02", checkpointResponse.Items[0])
	if err != nil {
		t.Error("Error consuming kafka message")
	}
}
func consumeMessage(key string, value string) error {
	// set the message whose key is "kubenode02" and value is kubeletResponse.Items[0] as an environment variable
	// consume the message from the kafka broker
	value = util.ModifyCheckpointToImageName(value)
	
	bootstrapServers := "192.168.56.3:32195"
    
    topic := "my-topic"
    groupID := "my-group"
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
            return err
        }
        if string(msg.Key) == key && string(msg.Value) == value {
            // Commit the offset to acknowledge the message has been processed
            if err := reader.CommitMessages(context.Background(), msg); err != nil {
                log.Fatalf("Failed to commit message: %v", err)
            }
            return nil
        }
    }
}

func produceMessage(key string, value string) error {
	//set the message whose key is "kubenode02" and value is kubeletResponse.Items[0] as an environment variable
	value = util.ModifyCheckpointToImageName(value)
	bootstrapServers := "192.168.56.3:32195"

	topic := "my-topic"
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},	
	}
	message := kafka.Message{
		// Key:   []byte(key), // Optional: specify a key for the message
		// Value: []byte(value),
		Key:  []byte(key), // Optional: specify a key for the message
		Value: []byte(value),
	}
	// Send the message
	err := writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}

	fmt.Println("Message sent successfully.")

	// Close the producer
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}
