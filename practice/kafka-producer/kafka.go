package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"


	"github.com/segmentio/kafka-go"
)
// This file is for testing strimzi kafka server, not for production
func main() {
	// bootstrapServers := "my-cluster-kafka-bootstrap:9092"
	// bootstrapServers := "bootstrap.my-kafka.example.com:31192"
	bootstrapServers := "192.168.56.3:32195"
	topic := "my-topic"
	// filePath := "/home/Tony/MyOperstorProjects/practice/yaml/kafka/ca.crt"
	// tlsConfig := createConfigFromCA(filePath)

	// Create a Kafka producer
	writer := kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		// Transport: &kafka.Transport{
		// 	TLS: tlsConfig,
		// },
	}

	// key := os.Getenv("kafka-key")
	// value := os.Getenv("kafka-value")

	// Prepare the message,key and value are read from environment variables
	message := kafka.Message{
		// Key:   []byte(key), // Optional: specify a key for the message
		// Value: []byte(value),
		Key:  []byte("key1"), // Optional: specify a key for the message
		Value: []byte("value1"),
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
func createConfigFromCA(filePath string) *tls.Config {
	// Load the CA certificate
	caCert, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}
	// Create a certificate pool and add the CA certificate
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create a TLS configuration that uses the CA certificate
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	return tlsConfig

	// Now you can use the tlsConfig for secure connections
}
