package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/pflag"
)

// flags
var (
	kubeconfig string
	// fetch this item from 1password:
	vault string
	item  string
	// and create this secret in kubernetes:
	namespace string
	secret    string
)

func init() {
	if home := homedir.HomeDir(); home != "" {
		pflag.StringVarP(&kubeconfig, "kubeconfig", "k", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		pflag.StringVarP(&kubeconfig, "kubeconfig", "k", "", "absolute path to the kubeconfig file")
	}
	pflag.StringVarP(&vault, "vault", "v", "", "1password vault name")
	pflag.StringVarP(&item, "item", "i", "", "1password item name")
	pflag.StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace for secret")
	pflag.StringVarP(&secret, "secret", "s", "", "kubernetes secret name")
	pflag.Parse()
	if kubeconfig == "" {
		log.Fatalln("please specify a kubernetes config (--kubeconfig)")
	}
	if vault == "" || item == "" {
		log.Fatalln("please specify a vault and item to extract (--vault && --item)")
	}
	if namespace == "" || secret == "" {
		log.Fatalln("please specify and namespace and secret to create (--namespace && --secret)")
	}
}

func main() {

	// first make sure we can connect to 1password
	cmd := exec.Command("op", "get", "account")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to successfully run \"op get account\": %s\n", err)
	}
	json, err := gabs.ParseJSON(stdout.Bytes())
	if err != nil {
		log.Fatalf("failed to parse json response from \"op get account\"\n")
	}
	// this is just a sanity check that we are getting the right sort of data back
	// we don't actually need either of these fields
	if !json.Exists("name") || !json.Exists("uuid") {
		log.Fatalf("failed to determine account info from \"op get account\"\n")
	}

	// now make sure we can connect to kubernetes
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("failed to build kubernetes config: %s\n", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to connect to kubernetes: %s\n", err.Error())
	}

	// make sure the namespace is there, and the secret is not
	_, err = clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("namespace \"%s\" does not exist; create first\n", namespace)
	}
	_, err = clientset.CoreV1().Secrets(namespace).Get(secret, metav1.GetOptions{})
	if err == nil {
		log.Fatalf("secret \"%s/%s\" exists already; delete first\n", namespace, secret)
	}

	// query the item in 1password to extract the fields
	secretMap := make(map[string]string)

	stdout.Reset()
	cmd = exec.Command("op", "get", "item", fmt.Sprintf("--vault=%s", vault), item)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("failed to extract item from 1password: %s\n", err.Error())
	}
	json, err = gabs.ParseJSON(stdout.Bytes())
	if err != nil {
		log.Fatalf("failed to parse json response from \"op get item\"\n")
	}
	fields, _ := json.Path("details.fields").Children()
	for _, field := range fields {
		if field.Exists("name") && field.Exists("value") {
			secretMap[field.Path("name").Data().(string)] = field.Path("value").Data().(string)
		}
	}
	sections, _ := json.Path("details.sections").Children()
	for _, section := range sections {
		fields, _ := section.Path("fields").Children()
		for _, field := range fields {
			if field.Exists("t") && field.Exists("v") {
				secretMap[field.Path("t").Data().(string)] = field.Path("v").Data().(string)
			}
		}
	}
	urls, _ := json.Path("overview.URLs").Children()
	for _, url := range urls {
		if url.Exists("l") && url.Exists("u") {
			secretMap[url.Path("l").Data().(string)] = url.Path("u").Data().(string)
		}
	}

	if len(secretMap) == 0 {
		log.Fatalf("1password item didn't give up any secrets!")
	}

	// create a new secret object
	secretObj := &apiv1.Secret{
		Type: "Opaque",
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret,
			Namespace: namespace,
		},
		StringData: secretMap,
	}
	// and create it in the cluster...
	secretObj, err = clientset.CoreV1().Secrets(namespace).Create(secretObj)
	if err != nil {
		log.Fatalf("failed to create secret \"%s/%s\": %s\n", namespace, secret, err.Error())
	} else {
		log.Printf("secret \"%s/%s\" created\n", namespace, secret)
	}
	spew.Dump(secretObj)

}
