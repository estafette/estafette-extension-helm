package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"gopkg.in/yaml.v2"
)

var (
	appgroup  string
	app       string
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	paramsYAML = kingpin.Flag("params-yaml", "Extension parameters, created from custom properties.").Envar("ESTAFETTE_EXTENSION_CUSTOM_PROPERTIES_YAML").Required().String()

	gitName      = kingpin.Flag("git-name", "Repository name, used as application name if not passed explicitly and app label not being set.").Envar("ESTAFETTE_GIT_NAME").String()
	appLabel     = kingpin.Flag("app-name", "App label, used as application name if not passed explicitly.").Envar("ESTAFETTE_LABEL_APP").String()
	buildVersion = kingpin.Flag("build-version", "Version number, used if not passed explicitly.").Envar("ESTAFETTE_BUILD_VERSION").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// log startup message
	log.Printf("Starting % version %v...", app, version)

	log.Printf("Unmarshalling parameters / custom properties...")
	var params params
	err := yaml.Unmarshal([]byte(*paramsYAML), &params)
	if err != nil {
		log.Fatal("Failed unmarshalling parameters: ", err)
	}

	log.Printf("Setting defaults for parameters that are not set in the manifest...")
	params.SetDefaults(*gitName, *appLabel, *buildVersion)

	switch params.Action {
	case
		"lint":
		log.Printf("Linting chart %v...", params.Chart)
		runCommand("helm lint %v", params.Chart)

	case "package":
		log.Printf("Packaging chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)
		runCommand("helm package --save=false --app-version %v --version %v %v", params.AppVersion, params.Version, params.Chart)

	case "test":
		log.Printf("Testing chart %v with app version %v and version %v on kind host %v...", params.Chart, params.AppVersion, params.Version, params.KindHost)

		log.Printf("\nWaiting for kind host to be ready...\n")
		httpClient := &http.Client{
			Timeout: time.Second * 1,
		}

		for true {
			_, err := httpClient.Get(fmt.Sprintf("http://%v:10080/kubernetes-ready", params.KindHost))
			if err == nil {
				break
			} else {
				time.Sleep(1 * time.Second)
			}
		}

		log.Printf("\nPreparing kind host for using Helm...\n")
		response, err := httpClient.Get(fmt.Sprintf("http://%v:10080/config", params.KindHost))
		if err != nil {
			log.Fatalf("Failed to retrieve kind config from http://%v:10080/config; %v", params.KindHost, err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			log.Fatalf("Failed to retrieve kind config from http://%v:10080/config; status code %v", params.KindHost, response.StatusCode)
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Failed to retrieve kind config from http://%v:10080/config; %v", params.KindHost, err)
		}
		kubeConfig := strings.ReplaceAll(string(body), "localhost", params.KindHost)

		usr, _ := user.Current()
		homeDir := usr.HomeDir
		err = ioutil.WriteFile(filepath.Join(homeDir, ".kube/config"), []byte(kubeConfig), 0644)
		if err != nil {
			log.Fatal("Failed writing ~/.kube/config: ", err)
		}

		runCommand("kubectl -n kube-system create serviceaccount tiller")
		runCommand("kubectl create clusterrolebinding tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller")
		runCommand("helm init --service-account tiller --wait")

		filesParameter := ""
		if params.Values != "" {
			log.Printf("\nWriting values to values.yaml...\n")
			err = ioutil.WriteFile("values.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal("Failed writing values.yaml: ", err)
			}
			filesParameter = "-f values.yaml"
			runCommand("cat values.yaml")
		}

		log.Printf("\nShowing template to be installed...\n")
		runCommand("helm diff upgrade %v %v-%v.tgz %v", params.Chart, params.Chart, params.Version, filesParameter)

		log.Printf("\nInstalling chart and waiting for %vs for it to be ready...\n", params.Timeout)
		err = runCommandExtended("helm upgrade --install %v %v-%v.tgz %v --wait --timeout %v", params.Chart, params.Chart, params.Version, filesParameter, params.Timeout)
		if err != nil {
			log.Printf("Installation timed out, showing logs...")
			runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v", params.Chart, params.Chart)
			os.Exit(1)
		}

		log.Printf("\nShowing logs for container...\n")
		runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v", params.Chart, params.Chart)

	case "publish":
		log.Printf("Publishing chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		runCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.ChartsSubdirectory)
		runCommand("cp %v-%v.tgz %v/%v", params.Chart, params.Version, params.RepositoryDirectory, params.ChartsSubdirectory)
		err = os.Chdir(params.RepositoryDirectory)
		if err != nil {
			log.Fatalf("Failed changing directory to %v; %v", params.RepositoryDirectory, err)
		}

		log.Printf("\nGenerating/updating index file for repository %v...\n", params.ChartsRepositoryURL)
		runCommand("helm repo index --url %v .", params.ChartsRepositoryURL)

		log.Printf("\nPushing changes to repository...\n")
		runCommandWithArgs("git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
		runCommandWithArgs("git", []string{"config", "--global", "user.name", "'estafette-bot'"})
		runCommand("git add --all")
		runCommandWithArgs("git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'%v v%v'", params.Chart, params.Version)})
		runCommand("git push origin master")

	case "purge":
		log.Printf("Purging pre-release version for chart %v with versions '%v-.+'...", params.Chart, params.Version)

		runCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.ChartsSubdirectory)
		err = os.Chdir(params.RepositoryDirectory)
		if err != nil {
			log.Fatalf("Failed changing directory to %v; %v", params.RepositoryDirectory, err)
		}

		filesGlob := fmt.Sprintf("%v/%v/%v-%v-*.tgz", params.RepositoryDirectory, params.ChartsSubdirectory, params.Chart, params.Version)
		files, err := filepath.Glob(filesGlob)
		if err != nil {
			log.Fatalf("Failed globbing %v; %v", filesGlob, err)
		}
		runCommand("rm -f %v", strings.Join(files, " "))

		log.Printf("\nGenerating/updating index file for repository %v...\n", params.ChartsRepositoryURL)
		runCommand("helm repo index --url %v .", params.ChartsRepositoryURL)

		log.Printf("\nPushing changes to repository...\n")
		runCommandWithArgs("git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
		runCommandWithArgs("git", []string{"config", "--global", "user.name", "'estafette-bot'"})
		runCommand("git add --all")
		runCommandWithArgs("git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'purged %v v%v-.+'", params.Chart, params.Version)})
		runCommand("git push origin master")

	case "install":
		log.Printf("Install chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		// TODO get kube config for target to deploy to

		filesParameter := ""
		if params.Values != "" {
			log.Printf("\nWriting values to values.yaml...\n")
			err = ioutil.WriteFile("values.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal("Failed writing values.yaml: ", err)
			}
			filesParameter = "-f values.yaml"
			runCommand("cat values.yaml")
		}

		log.Printf("\nShowing template to be installed...\n")
		err = runCommandExtended("helm diff upgrade %v %v-%v.tgz %v -n %v", params.ReleaseName, params.Chart, params.Version, filesParameter, params.Namespace)

		log.Printf("\nInstalling chart and waiting for %vs for it to be ready...\n", params.Timeout)
		err = runCommandExtended("helm upgrade --install %v %v-%v.tgz %v -n %v --wait --timeout %v", params.ReleaseName, params.Chart, params.Version, filesParameter, params.Namespace, params.Timeout)
		if err != nil {
			log.Printf("Installation timed out, showing logs...")
			runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v,app.kubernetes.io/version=%v -n %v", params.Chart, params.ReleaseName, params.Version, params.Namespace)
			os.Exit(1)
		}

		log.Printf("\nShowing logs for container...\n")
		runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v,app.kubernetes.io/version=%v -n %v", params.Chart, params.ReleaseName, params.Version, params.Namespace)
	default:
		log.Fatalf("Action '%v' is not supported; please use action parameter value 'lint','package','test', 'publish', 'install' or 'purge'", params.Action)
	}

}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func runCommand(command string, args ...interface{}) {
	err := runCommandExtended(command, args...)
	handleError(err)
}

func runCommandExtended(command string, args ...interface{}) error {
	command = fmt.Sprintf(command, args...)
	commandArray := strings.Split(command, " ")
	var c string
	var a []string
	if len(commandArray) > 0 {
		c = commandArray[0]
	}
	if len(commandArray) > 1 {
		a = commandArray[1:]
	}

	log.Printf("> %v %v", c, strings.Join(a, " "))
	cmd := exec.Command(c, a...)
	cmd.Dir = "/estafette-work"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}

func runCommandWithArgs(command string, args []string) {
	err := runCommandWithArgsExtended(command, args)
	handleError(err)
}

func runCommandWithArgsExtended(command string, args []string) error {
	log.Printf("> %v %v", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = "/estafette-work"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}
