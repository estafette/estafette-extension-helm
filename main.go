package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
		log.Printf("Packaging chart $chart with app version %v and version %v...", params.AppVersion, params.Version)
		runCommand("helm package --save=false --app-version %v --version %v %v", params.AppVersion, params.Version, params.Chart)

	case "test":
		log.Printf("Testing chart $chart with app version $appversion and version $version on kind host $kindhost...")

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
		err = ioutil.WriteFile("~/.kube/config", []byte(kubeConfig), 0600)
		if err != nil {
			log.Fatal("Failed writing ~/.kube/config: ", err)
		}

		runCommand("kubectl -n kube-system create serviceaccount tiller")
		runCommand("kubectl create clusterrolebinding tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller")
		runCommand("helm init --service-account tiller --wait")

		setParameters := params.GetSetParameters()
		if setParameters != "" {
			log.Printf("Using following arguments for setting values:\n%v", setParameters)
		}

		log.Printf("\nShowing template to be installed...\n")
		runCommand("helm template --name %v %v-%v.tgz %v", params.Chart, params.Chart, params.Version, setParameters)

		log.Printf("\nInstalling chart and waiting for ${timeout}s for it to be ready...\n")
		err = runCommandExtended("helm upgrade --install %v %v-%v.tgz %v --wait --timeout %v", params.Chart, params.Chart, params.Version, setParameters, params.Timeout)
		if err != nil {
			log.Printf("Installation timed out, showing logs...")
			runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v", params.Chart, params.Chart)
			os.Exit(1)
		}

		log.Printf("\nShowing logs for container...\n")
		runCommand("kubectl logs -l app.kubernetes.io/name=%v,app.kubernetes.io/instance=%v", params.Chart, params.Chart)

	case "publish":
		log.Printf("Publishing chart $chart with app version $appversion and version $version...")

		runCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.ChartsSubdirectory)
		runCommand("cp *.tgz %v/%v", params.RepositoryDirectory, params.ChartsSubdirectory)
		runCommand("cd %v", params.RepositoryDirectory)

		log.Printf("\nGenerating/updating index file for repository $repourl...\n")
		runCommand("helm repo index --url %v .", params.ChartsRepositoryURL)

		log.Printf("\nPushing changes to repository...\n")
		runCommand("git config --global user.email 'bot@estafette.io'")
		runCommand("git config --global user.name 'Estafette bot'")
		runCommand("git add --all")
		runCommand("git commit --allow-empty -m '%v v%v'", params.Chart, params.Version)
		runCommand("git push origin master")
	case "purge":
		log.Printf("Purging pre-release version for chart $chart with versions '$version-.+'...")

		runCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.ChartsSubdirectory)
		runCommand("cd %v", params.RepositoryDirectory)
		runCommand("rm -f %v/%v/%v-%v-*.tgz", params.RepositoryDirectory, params.ChartsSubdirectory, params.Chart, params.Version)

		log.Printf("\nGenerating/updating index file for repository $repourl...\n")
		runCommand("helm repo index --url %v .", params.ChartsRepositoryURL)

		log.Printf("\nPushing changes to repository...\n")
		runCommand("git config --global user.email 'bot@estafette.io'")
		runCommand("git config --global user.name 'Estafette bot'")
		runCommand("git add --all")
		runCommand("git commit --allow-empty -m 'purged ${chart} v${version}-.+'", params.Chart, params.Version)
		runCommand("git push origin master")

	default:
		log.Fatal("Action '$ESTAFETTE_EXTENSION_ACTION' is not supported; please use action parameter value 'lint','package','test', 'publish' or 'purge'")
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

	log.Printf("Running command '%v %v'...", c, strings.Join(a, " "))
	cmd := exec.Command(c, a...)
	cmd.Dir = "/estafette-work"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}
