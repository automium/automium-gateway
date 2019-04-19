package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	types "github.com/automium/types/go/gateway"
	"github.com/getsentry/raven-go"
	"github.com/ghodss/yaml"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func init() {
	raven.SetDSN("https://9975f03f6b28410d821116a322c8b678:12b01191ebe14165ae65244a268a0eae@stacktracer.enter.it/3")
}

//TODO: move to the shared types lib
type GitSecret struct {
	GitConfig types.GitConfig `json:"git"`
}

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
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("[ERROR] Invalid input: %s", err.Error())
	}

	secretBytes, err := getAPISecret("GitConfig")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	var gitSecret GitSecret
	err = json.Unmarshal(secretBytes, &gitSecret)

	var inputData = gitSecret.GitConfig
	err = validateData(inputData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("[ERROR] Invalid data: %s", err.Error())
	}

	// Parse SSH key from input
	sshKey, err := ssh.ParsePrivateKey([]byte(inputData.RepositoryKey))
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("[ERROR] Invalid SSH key: %s", err.Error())
	}

	// Clones the given repository, creating the remote, the local branches
	// and fetching the objects, everything in memory:
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth: returnSSHConfiguration(inputData.RepositoryUsername, sshKey),
		URL:  inputData.RepositoryURL,
	})
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err.Error())
	}

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err.Error())
	}

	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err.Error())
	}

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err.Error())
	}

	// ... get the files iterator and return the content as json
	var output = "["
	tree.Files().ForEach(func(f *object.File) error {
		content, err := f.Contents()
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			log.Fatal(err.Error())
		}
		spec, err := yaml.YAMLToJSON([]byte(content))
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			log.Fatal(err.Error())
		}
		output += fmt.Sprintf("%s,", string(spec))
		return nil
	})
	//remove the last comma
	sz := len(output)
	if sz > 0 && output[sz-1] == ',' {
		output = output[:sz-1]
	}
	output += "]"
	return fmt.Sprintf(output)
}

func validateInput(input string) error {
	//log.Printf("request with %s key", input)
	// TODO: validation
	return nil
}

func validateData(input types.GitConfig) error {
	// TODO: validation
	return nil
}

func returnSSHConfiguration(user string, signer ssh.Signer) *gitssh.PublicKeys {
	obj := &gitssh.PublicKeys{User: user, Signer: signer}
	// TODO: find a way to check SSH host keys
	obj.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	return obj
}
