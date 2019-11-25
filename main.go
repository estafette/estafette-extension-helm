package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	foundation "github.com/estafette/estafette-foundation"
	zerolog "github.com/rs/zerolog/log"
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

	gitName           = kingpin.Flag("git-name", "Repository name, used as application name if not passed explicitly and app label not being set.").Envar("ESTAFETTE_GIT_NAME").String()
	appLabel          = kingpin.Flag("app-name", "App label, used as application name if not passed explicitly.").Envar("ESTAFETTE_LABEL_APP").String()
	buildVersion      = kingpin.Flag("build-version", "Version number, used if not passed explicitly.").Envar("ESTAFETTE_BUILD_VERSION").String()
	releaseTargetName = kingpin.Flag("release-target-name", "Name of the release target, which is used by convention to resolve the credentials.").Envar("ESTAFETTE_RELEASE_NAME").String()
	credentialsJSON   = kingpin.Flag("credentials", "GKE credentials configured at service level, passed in to this trusted extension.").Envar("ESTAFETTE_CREDENTIALS_KUBERNETES_ENGINE").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// init log format from envvar ESTAFETTE_LOG_FORMAT
	foundation.InitLoggingFromEnv(appgroup, app, version, branch, revision, buildDate)

	zerolog.Info().Msg("Unmarshalling parameters / custom properties...")
	var params params
	err := yaml.Unmarshal([]byte(*paramsYAML), &params)
	if err != nil {
		log.Fatal("Failed unmarshalling parameters: ", err)
	}

	zerolog.Info().Msg("Setting defaults for parameters that are not set in the manifest...")
	params.SetDefaults(*gitName, *appLabel, *buildVersion, *releaseTargetName)

	switch params.Action {
	case
		"lint":
		zerolog.Info().Msgf("Linting chart %v...", params.Chart)
		foundation.RunCommand("helm lint %v", filepath.Join(params.HelmSubdirectory, params.Chart))

	case "package":
		zerolog.Info().Msgf("Packaging chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)
		foundation.RunCommand("helm package --app-version %v --version %v %v", params.AppVersion, params.Version, filepath.Join(params.HelmSubdirectory, params.Chart))

	case "test":
		zerolog.Info().Msgf("Testing chart %v with app version %v and version %v on kind host %v...", params.Chart, params.AppVersion, params.Version, params.KindHost)

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

		overrideValuesFilesParameter := ""
		if params.Values != "" {
			log.Printf("\nWriting values to override.yaml...\n")
			err = ioutil.WriteFile("override.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal("Failed writing override.yaml: ", err)
			}
			overrideValuesFilesParameter = "-f override.yaml"
			foundation.RunCommand("cat override.yaml")
		}

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		log.Printf("\nShowing template to be installed...\n")
		foundation.RunCommand("helm diff upgrade %v %v %v --allow-unreleased", params.Chart, filename, overrideValuesFilesParameter)

		log.Printf("\nInstalling chart file %v and waiting for %vs for it to be ready...\n", filename, *params.Timeout)
		err = foundation.RunCommandExtended("helm upgrade --install %v %v %v --wait --timeout %v", params.Chart, filename, overrideValuesFilesParameter, *params.Timeout)

		if err != nil {
			log.Printf("Installation failed, showing logs...")
			foundation.RunCommand("kubectl get all")
			foundation.RunCommand("kubectl logs -l app.kubernetes.io/instance=%v --all-containers=true", params.Chart)
			os.Exit(1)
		}

		log.Printf("\nShowing logs for container...\n")
		foundation.RunCommand("kubectl logs -l app.kubernetes.io/instance=%v --all-containers=true", params.Chart)

	case "publish":
		log.Printf("Publishing chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		foundation.RunCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
		foundation.RunCommand("cp %v %v/%v", filename, params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
		err = os.Chdir(params.RepositoryDirectory)
		if err != nil {
			log.Fatalf("Failed changing directory to %v; %v", params.RepositoryDirectory, err)
		}

		log.Printf("\nGenerating/updating index file for repository %v...\n", params.RepositoryURL)
		foundation.RunCommand("helm repo index --url %v .", params.RepositoryURL)

		log.Printf("\nPushing changes to repository...\n")
		foundation.RunCommandWithArgs("git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
		foundation.RunCommandWithArgs("git", []string{"config", "--global", "user.name", "'estafette-bot'"})
		foundation.RunCommand("git status")
		foundation.RunCommand("git add --all")
		foundation.RunCommandWithArgs("git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'%v v%v'", params.Chart, params.Version)})
		foundation.RunCommand("git push origin master")

	case "purge":
		zerolog.Info().Msgf("Purging pre-release version for chart %v with versions '%v-.+'...", params.Chart, params.Version)

		foundation.RunCommand("mkdir -p %v/%v", params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
		err = os.Chdir(params.RepositoryDirectory)
		if err != nil {
			log.Fatalf("Failed changing directory to %v; %v", params.RepositoryDirectory, err)
		}

		filesGlob := fmt.Sprintf("%v/%v-%v-*.tgz", params.RepositoryChartsSubdirectory, params.Chart, params.Version)
		log.Printf("glob: %v", filesGlob)
		files, err := filepath.Glob(filesGlob)
		if err != nil {
			log.Fatalf("Failed globbing %v; %v", filesGlob, err)
		}
		if len(files) > 0 {
			foundation.RunCommand("rm -f %v", strings.Join(files, " "))

			log.Printf("\nGenerating/updating index file for repository %v...\n", params.RepositoryURL)
			foundation.RunCommand("helm repo index --url %v .", params.RepositoryURL)

			log.Printf("\nPushing changes to repository...\n")
			foundation.RunCommandWithArgs("git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
			foundation.RunCommandWithArgs("git", []string{"config", "--global", "user.name", "'estafette-bot'"})
			foundation.RunCommand("git add --all")
			foundation.RunCommandWithArgs("git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'purged %v v%v-.+'", params.Chart, params.Version)})
			foundation.RunCommand("git push origin master")

		} else {
			log.Printf("Found 0 files to purge")
		}

	case "diff", "install":
		zerolog.Info().Msgf("Installing chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		if *credentialsJSON == "" {
			log.Fatal("Credentials of type kubernetes-engine are not injected; configure this extension as trusted and inject credentials of type kubernetes-engine")
		}

		log.Printf("Unmarshalling injected credentials...")
		var credentials []GKECredentials
		err = json.Unmarshal([]byte(*credentialsJSON), &credentials)
		if err != nil {
			log.Fatal("Failed unmarshalling injected credentials: ", err)
		}

		log.Printf("Checking if credential %v exists...", params.Credentials)
		credential := GetCredentialsByName(credentials, params.Credentials)
		if credential == nil {
			log.Fatalf("Credential with name %v does not exist.", params.Credentials)
		}

		log.Printf("Retrieving service account email from credentials...")
		var keyFileMap map[string]interface{}
		err = json.Unmarshal([]byte(credential.AdditionalProperties.ServiceAccountKeyfile), &keyFileMap)
		if err != nil {
			log.Fatal("Failed unmarshalling service account keyfile: ", err)
		}
		var saClientEmail string
		if saClientEmailIntfc, ok := keyFileMap["client_email"]; !ok {
			log.Fatal("Field client_email missing from service account keyfile")
		} else {
			if t, aok := saClientEmailIntfc.(string); !aok {
				log.Fatal("Field client_email not of type string")
			} else {
				saClientEmail = t
			}
		}

		log.Printf("Storing gke credential %v on disk...", params.Credentials)
		err = ioutil.WriteFile("/key-file.json", []byte(credential.AdditionalProperties.ServiceAccountKeyfile), 0600)
		if err != nil {
			log.Fatal("Failed writing service account keyfile: ", err)
		}

		log.Printf("Authenticating to google cloud")
		foundation.RunCommandWithArgs("gcloud", []string{"auth", "activate-service-account", saClientEmail, "--key-file", "/key-file.json"})

		log.Printf("Setting gcloud account to %v", saClientEmail)
		foundation.RunCommandWithArgs("gcloud", []string{"config", "set", "account", saClientEmail})

		log.Printf("Setting gcloud project")
		foundation.RunCommandWithArgs("gcloud", []string{"config", "set", "project", credential.AdditionalProperties.Project})

		log.Printf("Getting gke credentials for cluster %v", credential.AdditionalProperties.Cluster)
		clustersGetCredentialsArsgs := []string{"container", "clusters", "get-credentials", credential.AdditionalProperties.Cluster}
		if credential.AdditionalProperties.Zone != "" {
			clustersGetCredentialsArsgs = append(clustersGetCredentialsArsgs, "--zone", credential.AdditionalProperties.Zone)
		} else if credential.AdditionalProperties.Region != "" {
			clustersGetCredentialsArsgs = append(clustersGetCredentialsArsgs, "--region", credential.AdditionalProperties.Region)
		} else {
			log.Fatal("Credentials have no zone or region; at least one of them has to be defined")
		}
		foundation.RunCommandWithArgs("gcloud", clustersGetCredentialsArsgs)

		overrideValuesFilesParameter := ""
		if params.Values != "" {
			log.Printf("\nWriting values to override.yaml...\n")
			err = ioutil.WriteFile("override.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal("Failed writing override.yaml: ", err)
			}
			overrideValuesFilesParameter = "-f override.yaml"
			foundation.RunCommand("cat override.yaml")
		}

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		if !foundation.FileExists(filename) {
			log.Printf("\nNo helm package present, retrieving helm chart %v version %v from %v...\n", params.Chart, params.Version, params.RepositoryURL)
			foundation.RunCommand("helm fetch %v --version %v --repo %v", params.Chart, params.Version, params.RepositoryURL)
		}

		log.Printf("\nShowing template to be installed...\n")
		foundation.RunCommand("helm diff upgrade %v %v %v --namespace %v --allow-unreleased", params.ReleaseName, filename, overrideValuesFilesParameter, params.Namespace)

		if params.Action == "install" {
			log.Printf("\nInstalling chart and waiting for %vs for it to be ready...\n", *params.Timeout)
			err = foundation.RunCommandExtended("helm upgrade --install %v %v %v --namespace %v --wait --timeout %v", params.ReleaseName, filename, overrideValuesFilesParameter, params.Namespace, *params.Timeout)
			if err != nil {
				log.Printf("Installation failed, showing logs...")
				foundation.RunCommand("kubectl get all -n %v", params.Namespace)
				foundation.RunCommand("kubectl logs -l app.kubernetes.io/instance=%v -n %v --all-containers=true", params.ReleaseName, params.Namespace)
				os.Exit(1)
			}

			log.Printf("\nShowing logs for container...\n")
			foundation.RunCommand("kubectl logs -l app.kubernetes.io/instance=%v -n %v --all-containers=true", params.ReleaseName, params.Namespace)
		}
	default:
		zerolog.Fatal().Msgf("Action '%v' is not supported; please use action parameter value 'lint', 'package', 'test', 'publish', 'diff', 'install' or 'purge'", params.Action)
	}
}
