package function

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	types "github.com/automium/types/go/gateway"
	v1beta1 "github.com/automium/types/go/v1beta1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getAPISecret(secretName string) (secretBytes []byte, err error) {
	root := "/var/openfaas/secrets/"

	// read from the openfaas secrets folder
	secretBytes, err = ioutil.ReadFile(root + secretName)
	return secretBytes, err
}

// Handle a serverless request
func Handle(req []byte) string {

	key := os.Getenv("Http_X_Api_Key")
	err := validateInput(key)
	if err != nil {
		log.Fatalf("[ERROR] Invalid input: %s", err.Error())
	}

	secretBytes, err := getAPISecret("KubeConfig")
	if err != nil {
		log.Fatal(err)
	}

	var inputData types.KubernetesConfig
	err = json.Unmarshal(secretBytes, &inputData)

	err = validateData(inputData)
	if err != nil {
		log.Fatalf("[ERROR] Invalid data: %s", err.Error())
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(inputData.Kubeconfig))
	if err != nil {
		log.Fatalf("[ERROR] Cannot create configuration from provided kubeconfig: %s", err.Error())
	}

	client, err := createRESTClient(config)
	if err != nil {
		log.Fatalf("[ERROR] Cannot prepare the client: %s", err.Error())
	}

	result := v1beta1.NodeList{}
	err = client.Get().Resource("nodes").Do().Into(&result)
	if err != nil {
		log.Fatalf("[ERROR] Cannot retrieve nodes: %s", err.Error())
	}

	nodeListJSON, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("[ERROR] Cannot marshal output: %s", err.Error())
	}

	return string(nodeListJSON)
}

func createRESTClient(config *rest.Config) (*rest.RESTClient, error) {
	v1beta1.AddToScheme(scheme.Scheme)

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1beta1.GroupName, Version: v1beta1.GroupVersion}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	rc, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func validateInput(input string) error {
	//log.Printf("request with %s key", input)
	// TODO: validation
	return nil
}

func validateData(input types.KubernetesConfig) error {
	// TODO: validation
	return nil
}
