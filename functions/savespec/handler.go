package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	types "github.com/automium/types/go/gateway"
	"github.com/ghodss/yaml"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func getAPISecret(secretName string) (secretBytes []byte, err error) {
	root := "/var/openfaas/secrets/"

	// read from the openfaas secrets folder
	secretBytes, err = ioutil.ReadFile(root + secretName)
	return secretBytes, err
}

//TODO: move to the shared types lib
type GitSecret struct {
	GitConfig types.GitConfig `json:"git"`
}

// Handle a serverless request
func Handle(req []byte) string {

	key := os.Getenv("Http_X_Api_Key")
	err := validateInput(key)
	if err != nil {
		log.Fatalf("[ERROR] Invalid input: %s", err.Error())
	}

	secretBytes, err := getAPISecret("GitConfig")
	if err != nil {
		log.Fatal(err)
	}

	var gitSecret GitSecret
	err = json.Unmarshal(secretBytes, &gitSecret)

	var inputData types.SaveSpec
	err = json.Unmarshal(req, &inputData)
	if err != nil {
		log.Fatalf("[ERROR] Cannot parse incoming data: %s", err.Error())
	}
	inputData.GitConfig = gitSecret.GitConfig

	err = validateData(inputData)
	if err != nil {
		log.Fatalf("[ERROR] Invalid data: %s", err.Error())
	}

	// Generate a working directory
	workingDirectoryPath := fmt.Sprintf("/home/app/gitrepo_%s", uuid.NewV4().String())
	err = os.Mkdir(workingDirectoryPath, 0700)
	if err != nil {
		log.Fatalf("[ERROR] Cannot prepare temporary working dir: %s", err.Error())
	}

	// Parse SSH key from input
	sshKey, err := ssh.ParsePrivateKey([]byte(inputData.GitConfig.RepositoryKey))
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Invalid SSH key: %s", err.Error())
	}

	// Clone the repo
	repo, err := git.PlainClone(workingDirectoryPath, false, &git.CloneOptions{
		Auth: returnSSHConfiguration(inputData.GitConfig.RepositoryUsername, sshKey),
		URL:  inputData.GitConfig.RepositoryURL,
	})
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot checkout Git repository: %s", err.Error())
	}

	service, err := json.Marshal(inputData.Service)
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot parse service spec: %s", err.Error())
	}

	spec, err := yaml.JSONToYAML([]byte(service))
	if err != nil {
		log.Fatalf("[ERROR] Cannot convert service spec to yaml: %s", err.Error())
	}

	// Create or update the file
	// TODO: improve logic
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s.yaml", workingDirectoryPath, strings.ToLower(inputData.ServiceName)), spec, 0600)
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot update file with spec: %s", err.Error())
	}

	// Retrieve the working tree
	workingTree, err := repo.Worktree()
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot move to working tree: %s", err.Error())
	}

	// Add the file
	_, err = workingTree.Add(fmt.Sprintf("%s.yaml", strings.ToLower(inputData.ServiceName)))
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot add file to commit: %s", err.Error())
	}

	// Commit the change
	_, err = workingTree.Commit(fmt.Sprintf("[AUTOMIUM] Update %s spec", inputData.ServiceName), &git.CommitOptions{Author: &object.Signature{
		Name:  "Automium Bot",
		Email: "automium-bot@automium.io",
		When:  time.Now(),
	}})
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot commit: %s", err.Error())
	}

	// Push the change to the remote repository
	err = repo.Push(&git.PushOptions{
		Auth: returnSSHConfiguration(inputData.GitConfig.RepositoryUsername, sshKey),
	})
	if err != nil {
		cleanup(workingDirectoryPath)
		log.Fatalf("[ERROR] Cannot push: %s", err.Error())
	}

	// Cleanup...
	cleanup(workingDirectoryPath)

	// ...and we're good to go!
	return fmt.Sprintf("{ \"status\": \"OK\"}")
}

func validateInput(input string) error {
	//log.Printf("request with %s key", input)
	// TODO: validation
	return nil
}

func validateData(input types.SaveSpec) error {
	// TODO: validation
	return nil
}

func cleanup(path string) {
	os.RemoveAll(path)
}

func returnSSHConfiguration(user string, signer ssh.Signer) *gitssh.PublicKeys {
	obj := &gitssh.PublicKeys{User: user, Signer: signer}
	// TODO: find a way to check SSH host keys
	obj.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	return obj
}
