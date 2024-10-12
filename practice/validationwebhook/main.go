// package main

// import (
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"net/http"

// 	v1 "k8s.io/api/admission/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/apimachinery/pkg/runtime/serializer"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/klog/v2"
// )

// type MigrationSpec struct {
// 	PodName              string   `json:"podname,omitempty"`
// 	Deployment           string   `json:"deployment,omitempty"`
// 	Namespace            string   `json:"namespace"`
// 	DestinationNode      string   `json:"destinationNode"`
// 	DestinationNamespace string   `json:"destinationNamespace"`
// 	Specify              []string `json:"specify"`
// }

// type serverParameters struct {
// 	port     int    // webhook server port
// 	certFile string // path to the x509 certificate for https
// 	keyFile  string // path to the x509 private key matching `CertFile`
// }

// var (
// 	parameters            serverParameters
// 	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
// )

// func admitMigration(w http.ResponseWriter, r *http.Request) {
// 	// Decode the admission review request
// 	klog.Info("Admitting migration")

// 	request := v1.AdmissionReview{}
// 	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
// 		klog.Error(err)
// 		http.Error(w, "could not decode admission review", http.StatusBadRequest)
// 		return
// 	}

// 	// Decode the admission review request
// 	admissionReview := v1.AdmissionReview{}
// 	if _, _, err := universalDeserializer.Decode(request.Request.Object.Raw, nil, &admissionReview); err != nil {
// 		klog.Error(err)
// 		http.Error(w, "could not decode admission review", http.StatusBadRequest)
// 		return
// 	}

// 	// Decode the custom resource spec
// 	var migrationSpec MigrationSpec
// 	if err := json.Unmarshal(admissionReview.Request.Object.Raw, &migrationSpec); err != nil {
// 		klog.Error(err)
// 		http.Error(w, "could not decode migration spec", http.StatusBadRequest)
// 		return
// 	}

// 	// Validation logic: if podname is set, deployment must be empty, and vice versa
// 	var response v1.AdmissionResponse
// 	if migrationSpec.PodName != "" && migrationSpec.Deployment != "" {
// 		// only keeps the deployment field
// 		migrationSpec.PodName = ""
// 		response = v1.AdmissionResponse{
// 			Allowed: false,
// 			Result: &metav1.Status{
// 				Message: "podname and deployment cannot be set at the same time",
// 			},
// 		}
// 	} else {
// 		response = v1.AdmissionResponse{
// 			Allowed: true,
// 		}
// 	}

// 	admissionReview.Response = &response
// 	admissionReview.Response.UID = admissionReview.Request.UID

// 	// Encode the admission review response
// 	if err := json.NewEncoder(w).Encode(admissionReview); err != nil {
// 		klog.Error(err)
// 		http.Error(w, "could not encode response body", http.StatusInternalServerError)
// 		return
// 	}
// }

// func main() {
// 	flag.IntVar(&parameters.port, "port", 8443, "Webhook server port.")
// 	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/server.crt", "File containing the x509 Certificate for HTTPS.")
// 	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/server.key", "File containing the x509 private key to --tlsCertFile.")
// 	flag.Parse()

// 	klog.Info("Starting the webhook server")
// 	http.HandleFunc("/validate", admitMigration)
// 	klog.Fatal(http.ListenAndServeTLS(fmt.Sprintf("localhost:%d", parameters.port), parameters.certFile, parameters.keyFile, http.DefaultServeMux))
// }

package main

import (
	"log"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"flag"
	"io/ioutil"
	"strconv"

	"errors"

	"k8s.io/api/admission/v1beta1"

	"encoding/json"

	apiv1 "k8s.io/api/core/v1"
)

type ServerParameters struct {
	port     int    // webhook server port
	certFile string // path to the x509 certificate for https
	keyFile  string // path to the x509 private key matching `CertFile`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var parameters ServerParameters

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

var config *rest.Config
var clientSet *kubernetes.Clientset

func main() {

	useKubeConfig := os.Getenv("USE_KUBECONFIG")
	kubeConfigFilePath := os.Getenv("KUBECONFIG")

	flag.IntVar(&parameters.port, "port", 8443, "Webhook server port.")
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	if len(useKubeConfig) == 0 {
		// default to service account in cluster token
		c, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		config = c
	} else {
		//load from a kube config
		var kubeconfig string

		if kubeConfigFilePath == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		} else {
			kubeconfig = kubeConfigFilePath
		}

		// fmt.Println("kubeconfig: " + kubeconfig)

		c, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		config = c
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	clientSet = cs

	test()
	http.HandleFunc("/", HandleRoot)
	http.HandleFunc("/mutate", HandleMutate)
	log.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(parameters.port), parameters.certFile, parameters.keyFile, nil))
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("HandleRoot!"))
}

func HandleMutate(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	err = ioutil.WriteFile("/tmp/request", body, 0644)
	if err != nil {
		panic(err.Error())
	}

	var admissionReviewReq v1beta1.AdmissionReview

	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		errors.New("malformed admission review: request is nil")
	}

	fmt.Printf("Type: %v \t Event: %v \t Name: %v \n",
		admissionReviewReq.Request.Kind,
		admissionReviewReq.Request.Operation,
		admissionReviewReq.Request.Name,
	)

	var pod apiv1.Pod

	err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod)

	if err != nil {
		fmt.Errorf("could not unmarshal pod on admission request: %v", err)
	}

	var patches []patchOperation

	labels := pod.ObjectMeta.Labels
	labels["example-webhook"] = "it-worked"

	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: labels,
	})

	patchBytes, err := json.Marshal(patches)

	if err != nil {
		fmt.Errorf("could not marshal JSON patch: %v", err)
	}

	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}

	admissionReviewResponse.Response.Patch = patchBytes

	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		fmt.Errorf("marshaling response: %v", err)
	}

	w.Write(bytes)

}
