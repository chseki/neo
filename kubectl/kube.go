package kubectl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// Kind defines a single kubernetes object
type Kind int

// Kubernetes object kind to interact with
const (
	Secret Kind = iota
	ConfigMap
	Deployment
)

// Kubectl defines some actions to do with a kubernetes object
type Kubectl interface {
	Copy(ns string)
}

// Objects available to interact with
type (
	secret struct {
		name string
	}
	configmap struct {
		name string
	}
	deployment struct {
		name string
	}
)

func (s *secret) Copy(namespace string) {
	if err := switchToNamespace(namespace); err != nil {
		log.Fatalln(err)
	}

	dir := homePath + namespace + "-" + s.name

	os.Mkdir(dir, 0744)
	os.Chdir(dir)

	if err := createManifests(s.name); err != nil {
		log.Fatalln(err)
	}

	info := color.New(color.FgHiGreen).SprintFunc()
	exec := color.New(color.FgHiBlue).SprintFunc()

	log.Printf("Manifests successfully createad by Neo in %s", info(dir))
	fmt.Printf("just execute %s in your desired namespace\n", exec("kubectl apply -f "+namespace+"-"+s.name))
}

func (c *configmap) Copy(namespace string) {
	if err := switchToNamespace(namespace); err != nil {
		log.Fatalln(err)
	}

	dir := homePath + namespace + "-" + c.name

	os.Mkdir(dir, 0744)
	os.Chdir(dir)

	if err := createManifests(c.name); err != nil {
		log.Fatalln(err)
	}

	info := color.New(color.FgHiGreen).SprintFunc()
	exec := color.New(color.FgHiBlue).SprintFunc()

	log.Printf("Manifests successfully createad by Neo in %s", info(dir))
	fmt.Printf("Execute %s in your desired namespace\n", exec("kubectl apply -f "+namespace+"-"+c.name))
}

func (d *deployment) Copy(namespace string) {
	if err := switchToNamespace(namespace); err != nil {
		log.Fatalln(err)
	}

	dir := homePath + namespace + "-" + d.name

	os.Mkdir(dir, 0744)
	os.Chdir(dir)

	if err := createManifests(d.name); err != nil {
		log.Fatalln(err)
	}

	info := color.New(color.FgHiGreen).SprintFunc()
	exec := color.New(color.FgHiBlue).SprintFunc()

	log.Printf("Manifests successfully createad by Neo in %s", info(dir))
	fmt.Printf("Execute %s in your desired namespace\n", exec("kubectl apply -f "+namespace+"-"+d.name))
}

var homePath string

// SecretYaml needs to be public because of yaml.v3 package
type secretYaml struct {
	APIVersion string      `yaml:"apiVersion"`
	Data       interface{} `yaml:"data"`
	Kind       string      `yaml:"kind"`
	Metadata   parameters  `yaml:"metadata"`
	TypeSecret string      `yaml:"type"`
}

// Parameters needs to be public because of yaml.v3 package
type parameters struct {
	Name string `yaml:"name"`
}

// Initialize function, get HOME path
func init() {
	homePath = os.Getenv("HOME") + "/"
}

// Factory bla
func Factory(object Kind) Kubectl {
	switch object {
	case 0:
		return &secret{
			name: "secrets",
		}
	case 1:
		return &configmap{
			name: "configmaps",
		}
	case 2:
		return &deployment{
			name: "deployments",
		}
	default:
		return nil
	}
}

func switchToNamespace(namespace string) error {
	cmd := exec.Command("kubectl", "config", "set-context", "--current", "--namespace", namespace)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	return err
}

func readObjects(object string) (objects []string, err error) {
	cmd := exec.Command("kubectl", "get", object, "-o", "name")

	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	kubeOutput := strings.TrimSpace(out.String())

	if kubeOutput == "" {
		info := color.New(color.FgHiRed).SprintFunc()
		log.Fatalf("Error: There is no %s object in this namespace", info(object))
	}

	objects = strings.Split(kubeOutput, "\n")

	return
}

func createManifests(object string) error {
	// get secrets name existent in namespace
	objects, err := readObjects(object)

	var wg sync.WaitGroup

	// starts worker pool to create YAML files concurrently
	for _, o := range objects {

		switch {
		case strings.Contains(o, "default"):
			continue
		case strings.Contains(o, "registry"):
			continue
		case strings.Contains(o, "pod-autoscaler-token"):
			continue
		case strings.TrimSpace(o) == "":
			continue
		default:
			{
				log.Printf("creating manifest for: %v\n", o)
				wg.Add(1)
				go worker(o, &wg)
			}
		}
	}

	wg.Wait()

	return err
}

// Goroutines are lightweight and running in concurrency
// get yaml output from some secret and creates a new well-formatted YAML file
// ignoring all unecessary metadata from original file such as namespace, created-at, last state and so on
func worker(name string, wg *sync.WaitGroup) {
	defer wg.Done()

	cmd := exec.Command("kubectl", "get", name, "-o", "yaml")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	switch {
	case strings.Contains(name, "secret"):
		createSecretYAML(out)
	default:
		log.Printf("Creation of %v YAML file not implemented yet", name)
	}
}

// write a YAML file in the current direcotry based on the SecretYaml struct
func createSecretYAML(secret bytes.Buffer) {
	s := secretYaml{}

	err := yaml.Unmarshal(secret.Bytes(), &s)
	if err != nil {
		log.Fatalln(err)
	}

	y, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalln(err)
	}

	ioutil.WriteFile(s.Metadata.Name+"-secret.yaml", y, 0644)
}
