package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var homePath string

// SecretYaml needs to be public because of yaml.v3 package
type SecretYaml struct {
	APIVersion string      `yaml:"apiVersion"`
	Data       interface{} `yaml:"data"`
	Kind       string      `yaml:"kind"`
	Metadata   Parameters  `yaml:"metadata"`
	TypeSecret string      `yaml:"type"`
}

// Parameters needs to be public because of yaml.v3 package
type Parameters struct {
	Name string `yaml:"name"`
}

// Cluster info
type cluster struct {
	env      string
	operator string
}

// Initialize function, get HOME path
func init() {
	homePath = os.Getenv("HOME") + "/"
}

func main() {
	// declaring flags
	kops := flag.String("kops", "", "environment cluster to executing script")
	eks := flag.String("eks", "", "eks cluster to executing script")
	fromNS := flag.String("from", "", "namespace to read secrets")
	toNS := flag.String("to", "", "namespace to apply secrets manifest")

	// try to parse flags
	flag.Parse()

	// if flag gets it default value exit with error
	if *fromNS == "" || *toNS == "" || *kops == "" || *eks == "" {
		log.Fatalln("Error: unknown command for go-copy.\nUsage: go-copy -kops=env -eks=env -from=namespace -to=namespace")
	}

	clsKops := cluster{env: *kops, operator: "kops"}
	switchToCluster(clsKops)

	err := readSecretsFromNamespace(*fromNS)
	if err != nil {
		log.Fatalln(err)
	}

	clsEks := cluster{env: *eks, operator: "eks"}
	switchToCluster(clsEks)

	err = writeSecretsToNamespace(*toNS)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Removing manifests from system...")
	if err := clean(); err != nil {
		log.Fatalln(err)
	}

	getSecrets(*toNS)
}

// put to stdout kubectl get secrets command in target namespace
func getSecrets(ns string) {
	log.Printf("Secrets Available in %v:\n", ns)
	cmd := exec.Command("kubectl", "get", "secrets")

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	fmt.Println(out.String())
}

// remove go-secrets directory which was used to apply all manifests created in read in operation
func clean() error {
	log.Println("Cleaning auxiliar directory...")
	cmd := exec.Command("rm", "-rf", "go-secrets")

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	return err
}

func switchToNamespace(namespace string) error {
	cmd := exec.Command("namespace", namespace)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	return err
}

func switchToCluster(cls cluster) {
	var cmd *exec.Cmd

	if cls.operator == "eks" {
		switch cls.env {
		case "stage":
			cmd = exec.Command("login_eks_stage")
		case "prod":
			cmd = exec.Command("login_eks_prod")
		default:
			log.Fatalln("wrong environment to eks, try:\neks-stage | eks-prod")
		}
	} else {
		switch cls.env {
		case "prod":
			cmd = exec.Command("login_prod")
		case "prod-sa":
			cmd = exec.Command("login_prod_sa")
		case "stage":
			cmd = exec.Command("login_stage")
		default:
			log.Fatalln("wrong environment to kops, try:\nprod | prod-sa | stage ")
		}
	}

	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}

// Apply all manifests in target namespace
// the manifests are in the folder go-secrets
func writeSecretsToNamespace(namespace string) error {
	log.Printf("Applying manifest secrets into %v\n", namespace)

	err := switchToNamespace(namespace)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "apply", "-f", "go-secrets")

	err = cmd.Run()

	return err
}

// Read all secrets in a namespace listing by name
// if some secrets contains the world "registry" or "default" its ignored
// if not, starts a go routine that proccess it in a worker pool
// also create the go-secrets folder to retain the yaml processed by worker
// the program stucks here untill all worker done its job
// They are being processed in concurrency.
func readSecretsFromNamespace(namespace string) error {
	log.Printf("Reading secrets from %v\n", namespace)
	// switch namespace if error, throws it
	err := switchToNamespace(namespace)
	if err != nil {
		return err
	}

	// get secrets name existent in namespace
	cmd := exec.Command("kubectl", "get", "secrets", "-o", "name")

	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	secrets := strings.Split(out.String(), "\n")

	os.Mkdir(homePath+"go-secrets/", 0744)
	os.Chdir(homePath + "go-secrets")

	var wg sync.WaitGroup

	// starts worker pool to create YAML files concurrently
	for _, s := range secrets {

		switch {
		case strings.Contains(s, "default"):
			continue
		case strings.Contains(s, "registry"):
			continue
		case strings.Contains(s, "pod-autoscaler-token"):
			continue
		case strings.TrimSpace(s) == "":
			continue
		default:
			{
				log.Printf("creating manifest for: %v\n", s)
				wg.Add(1)
				go worker(s, &wg)
			}
		}
	}

	wg.Wait()

	os.Chdir(homePath)

	return err
}

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

	createSecretYAML(out)
}

// write a YAML file in the current direcotry based on the SecretYaml struct
func createSecretYAML(secret bytes.Buffer) {
	s := SecretYaml{}

	_ = yaml.Unmarshal(secret.Bytes(), &s)

	y, _ := yaml.Marshal(s)

	ioutil.WriteFile(s.Metadata.Name+"-secret.yaml", y, 0644)
}
