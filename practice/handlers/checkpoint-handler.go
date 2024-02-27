package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var clientCache *http.Client

func GetKubeletClient() *http.Client {
	if clientCache != nil {
		return clientCache
	}
	// clientCertPrefix := "/etc/kubernetes/pki"
	clientCertPrefix := "/home/Tony/cka/deployment-env"
	clientCert, err := tls.LoadX509KeyPair(
		// fmt.Sprintf("%s/client.crt", clientCertPrefix),
		fmt.Sprintf("%s/apiserver-kubelet-client2.crt", clientCertPrefix),
		//fmt.Sprintf("%s/client.key", clientCertPrefix),
		fmt.Sprintf("%s/apiserver-kubelet-client2.key", clientCertPrefix),
	)
	if err != nil {
		log.Log.Error(err, "could not read client cert key pair")
	}
	certs := x509.NewCertPool()

	// We should really load this path dynamically as this depends on deep internals of kubernetes
	pemData, err := os.ReadFile(fmt.Sprintf("%s/ca2.crt", clientCertPrefix))
	if err != nil {
		log.Log.Error(err, "could not read ca file")
	}
	certs.AppendCertsFromPEM(pemData)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            certs,
			Certificates:       []tls.Certificate{clientCert},
		},
	}
	return &http.Client{Transport: tr}
}
