package function

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	types "github.com/automium/types/go/gateway"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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

	var inputData types.ServiceLogs
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

	// create the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("[ERROR] Cannot prepare the client: %s", err.Error())
	}

	pods, err := client.CoreV1().Pods("default").List(metav1.ListOptions{})
	if err != nil {
		log.Fatalf("[ERROR] Cannot get jobs list %s", err.Error())
	}
	var pod *corev1.Pod
	for _, p := range pods.Items {
		if strings.HasPrefix(p.Name, inputData.ServiceName) {
			pod = &p
			break
		}
	}
	if pod == nil {
		return "Service logs not found"
	}

	podLogOpts := corev1.PodLogOptions{}
	request := client.CoreV1().Pods("default").GetLogs(pod.Name, &podLogOpts)

	readCloser, err := request.Stream()
	if err != nil {
		log.Fatalf("[ERROR] Cannot get service logs for pod %s. %s", pod.Name, err.Error())
	}

	defer readCloser.Close()
	out, err := ioutil.ReadAll(readCloser)
	if err != nil {
		log.Fatalf("[ERROR] Cannot read service logs %s", err.Error())
	}
	return string(out)
}

func validateInput(input string) error {
	//log.Printf("request with %s key", input)
	// TODO: validation
	return nil
}

func validateData(input types.ServiceLogs) error {
	// TODO: validation
	return nil
}
