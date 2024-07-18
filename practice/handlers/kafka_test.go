package handlers

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"testing"
	"time"
	util "tony123.tw/util"
)

func TestProduceMessageToDifferentTopics(t *testing.T) {
	// newpod.Name=httpd, nodeName="kubenode01"
	ProduceMessageToDifferentTopics("httpd", "default", "kubenode01")
	newPodName, nameSpace, nodeName, err := consumeMessagehelper()
	if err != nil {
		t.Error("Error consuming kafka message")
	}
	if newPodName != "httpd" || nameSpace != "default" || nodeName != "kubenode01" {
		t.Error("Error in consuming kafka message")
	}
}
func consumeMessagehelper() (string, string, string, error) {
	// bootstrapServers := "my-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
	bootstrapServers := "192.168.56.4:32195"
	topic := "delete-pod"
	groupID := "my-group"

	// Create a Kafka consumer (reader)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{bootstrapServers},
		Topic:   topic,
		GroupID: groupID,
	})

	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Log.Info("ready to fetch message")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return "", "", "", fmt.Errorf("context deadline exceeded")
				}
				return "", "", "", err
			}
			reader.CommitMessages(ctx, msg)
			//the mesage sent by the original sender is key: podName, value: nameSpace/nodeName
			logger.Log.Info("msg.Key", string(msg.Key), "msg.Value", string(msg.Value))

			// seperate the string with left half string is namespace and right half string is nodeName
			newPodName := string(msg.Key)
			nameSpace := string(msg.Value)[0:strings.Index(string(msg.Value), "/")]
			nodeName := string(msg.Value)[strings.Index(string(msg.Value), "/")+1:]
			return newPodName, nameSpace, nodeName, nil
		}
	}
}

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
		Brokers: []string{bootstrapServers},
		Topic:   topic,
		GroupID: groupID,
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
		Key:   []byte(key), // Optional: specify a key for the message
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
