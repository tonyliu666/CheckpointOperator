package handlers

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var clientCache *http.Client

func GetKubeletClient() *http.Client {
	if clientCache != nil {
		return clientCache
	}
	clientCertPrefix := "/var/run/secrets/kubelet-certs"
	clientCAPrefix := "/var/run/secrets/kubernetes.io/serviceaccount"
	clientCert, err := tls.LoadX509KeyPair(
		fmt.Sprintf("%s/client.crt", clientCertPrefix),
		fmt.Sprintf("%s/client.key", clientCertPrefix),
	)
	if err != nil {
		log.Log.Error(err, "could not read client cert key pair")
		return nil
	}
	certs := x509.NewCertPool()

	// We should really load this path dynamically as this depends on deep internals of kubernetes
	pemData, err := os.ReadFile(fmt.Sprintf("%s/ca.crt", clientCAPrefix))

	if err != nil {
		log.Log.Error(err, "could not read ca file")
		return nil
	}
	certs.AppendCertsFromPEM(pemData)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            certs,
			Certificates:       []tls.Certificate{clientCert},
		},
	}
	return &http.Client{
		Transport: tr,
		Timeout:   10 * time.Minute,
	}
}

func CheckpointPod(client *http.Client, address string) (*http.Response, error) {
	logger := log.Log
	fullURL := fmt.Sprintf("%s?timeout=%d", address, 10*60*1000)
	// Create the request body
	requestBody := bytes.NewBuffer([]byte("{}"))

	// Create a new request
	req, err := http.NewRequest("POST", fullURL, requestBody)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	CheckpointStartTime := time.Now()
	// Send the request
	resp, err := client.Do(req)
	// set the timeout for kubelet checkpoint api, the timeout within the query
	// resp, err := client.Post(address, "application/json", strings.NewReader("{}"))
	CheckpointEndTime := time.Now()
	CheckpointDuration := CheckpointEndTime.Sub(CheckpointStartTime).Milliseconds()
	logger.Info("Checkpoint Duration: ", "Duration", CheckpointDuration)
	// err now is facing the problem that the status code is 401 unauthorized
	if err != nil {
		logger.Error(err, "unable to send the request")
		return nil, err
	}
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		logger.Error(err, "your request is unauthorized")
		return nil, fmt.Errorf("unable to checkpoint the container")
	}
	// check the response status code
	if resp.StatusCode != http.StatusOK {
		logger.Error(err, "unable to checkpoint the container")
		return nil, fmt.Errorf("unable to checkpoint the container")
	}
	return resp, nil
}
