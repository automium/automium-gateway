package function

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	types "github.com/automium/types/go/gateway"
	v1beta1 "github.com/automium/types/go/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	key := os.Getenv("Http_x_api_key")
	err := validateInput(key)
	if err != nil {
		log.Fatalf("[ERROR] Invalid input: %s", err.Error())
	}

	secretBytes, err := getAPISecret("KubeConfig")
	if err != nil {
		log.Fatal(err)
	}

	var kubeConfig types.KubernetesConfig
	err = json.Unmarshal(secretBytes, &kubeConfig)

	var inputData types.ApplyService
	err = json.Unmarshal(req, &inputData)
	if err != nil {
		log.Fatalf("[ERROR] Cannot handle input data: %s", err.Error())
	}
	inputData.Kubeconfig = kubeConfig.Kubeconfig

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

	var result = v1beta1.Service{}
	var service = &v1beta1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: inputData.Service.APIVersion,
			Kind:       inputData.Service.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   inputData.Service.Metadata.Name,
			Labels: map[string]string{"app": inputData.Service.Metadata.Labels.App},
		},
		Spec: v1beta1.ServiceSpec{
			Replicas: inputData.Service.Spec.Replicas,
			Flavor:   inputData.Service.Spec.Flavor,
			Version:  inputData.Service.Spec.Version,
			Tags:     inputData.Service.Spec.Tags,
			Env:      inputData.Service.Spec.Env,
		},
	}
	err = client.Post().Resource("services").Namespace("default").Body(service).Do().Into(&result)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			err = client.Get().Resource("services").Name(inputData.Service.Metadata.Name).Namespace("default").Do().Into(&result)
			if err != nil {
				log.Fatalf("[ERROR] Cannot get service: %s", err.Error())
			}
			service.ObjectMeta.ResourceVersion = result.ObjectMeta.ResourceVersion
			err = client.Put().Resource("services").Name(inputData.Service.Metadata.Name).Namespace("default").Body(service).Do().Into(&result)
			if err != nil {
				log.Fatalf("[ERROR] Cannot update service: %s", err.Error())
			}
		} else {
			log.Fatalf("[ERROR] Cannot create service: %s", err.Error())
		}
	}

	serviceJSON, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("[ERROR] Cannot marshal output: %s", err.Error())
	}

	return string(serviceJSON)
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

func validateData(input types.ApplyService) error {
	// TODO: validation
	return nil
}
