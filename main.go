package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
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
	foundation.InitLoggingFromEnv(foundation.NewApplicationInfo(appgroup, app, version, branch, revision, buildDate))

	// create context to cancel commands on sigterm
	ctx := foundation.InitCancellationContext(context.Background())

	log.Info().Msg("Unmarshalling parameters / custom properties...")
	var params params
	err := yaml.Unmarshal([]byte(*paramsYAML), &params)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed unmarshalling parameters")
	}

	log.Info().Msg("Setting defaults for parameters that are not set in the manifest...")
	params.SetDefaults(*gitName, *appLabel, *buildVersion, *releaseTargetName)

	switch params.Action {
	case
		"lint":
		log.Info().Msgf("Linting chart %v...", params.Chart)
		foundation.RunCommand(ctx, "helm lint %v", filepath.Join(params.HelmSubdirectory, params.Chart))

	case "package":
		log.Info().Msgf("Packaging chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)
		foundation.RunCommand(ctx, "helm package --app-version %v --version %v %v", params.AppVersion, params.Version, filepath.Join(params.HelmSubdirectory, params.Chart))

	case "test":
		log.Info().Msgf("Testing chart %v with app version %v and version %v on kind host %v...", params.Chart, params.AppVersion, params.Version, params.KindHost)

		log.Info().Msg("Waiting for kind host to be ready...")
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

		log.Info().Msg("Preparing kind host for using Helm...")
		response, err := httpClient.Get(fmt.Sprintf("http://%v:10080/config", params.KindHost))
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to retrieve kind config from http://%v:10080/config", params.KindHost)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			log.Fatal().Msgf("Failed to retrieve kind config from http://%v:10080/config; status code %v", params.KindHost, response.StatusCode)
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to retrieve kind config from http://%v:10080/config", params.KindHost)
		}
		kubeConfig := strings.ReplaceAll(string(body), "localhost", params.KindHost)

		usr, _ := user.Current()
		homeDir := usr.HomeDir
		err = ioutil.WriteFile(filepath.Join(homeDir, ".kube/config"), []byte(kubeConfig), 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed writing ~/.kube/config")
		}

		overrideValuesFilesParameter := ""
		if params.Values != "" {
			log.Info().Msg("Writing values to override.yaml...")
			err = ioutil.WriteFile("override.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed writing override.yaml")
			}
			overrideValuesFilesParameter = "-f override.yaml"
			foundation.RunCommand(ctx, "cat override.yaml")
		}

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		log.Info().Msg("Showing template to be installed...")
		foundation.RunCommand(ctx, "helm diff upgrade %v %v %v --allow-unreleased", params.Chart, filename, overrideValuesFilesParameter)

		log.Printf("\nInstalling chart file %v and waiting for %v for it to be ready...\n", filename, params.Timeout)
		err = foundation.RunCommandExtended(ctx, "helm upgrade --install %v %v %v --history-max 1 --cleanup-on-fail --atomic --timeout %v", params.Chart, filename, overrideValuesFilesParameter, params.Timeout)

		if err != nil {
			log.Printf("Installation failed, showing logs...")
			foundation.RunCommand(ctx, "kubectl get all")
			_ = foundation.RunCommandExtended(ctx, "kubectl logs -l app.kubernetes.io/instance=%v --all-containers=true", params.Chart)
			os.Exit(1)
		}

		log.Info().Msg("Showing logs for container...")
		_ = foundation.RunCommandExtended(ctx, "kubectl logs -l app.kubernetes.io/instance=%v --all-containers=true", params.Chart)

	case "publish":
		log.Info().Msgf("Publishing chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		if params.Bucket != "" {
			// publish to gcs bucket
			initCredential(ctx, params)

			foundation.RunCommand(ctx, "helm repo add my-repository gs://%v", params.Bucket)
			foundation.RunCommand(ctx, "helm gcs push %v my-repository --service-account='/key-file.json' --retry --debug", filename)

		} else {
			// publish to git repo

			foundation.RunCommand(ctx, "mkdir -p %v/%v", params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
			foundation.RunCommand(ctx, "cp %v %v/%v", filename, params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
			err = os.Chdir(params.RepositoryDirectory)
			if err != nil {
				log.Fatal().Err(err).Msgf("Failed changing directory to %v", params.RepositoryDirectory)
			}

			log.Info().Msgf("Generating/updating index file for repository %v...", params.RepositoryURL)
			foundation.RunCommand(ctx, "helm repo index --url %v .", params.RepositoryURL)

			log.Info().Msg("Pushing changes to repository...")
			foundation.RunCommandWithArgs(ctx, "git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
			foundation.RunCommandWithArgs(ctx, "git", []string{"config", "--global", "user.name", "'estafette-bot'"})
			foundation.RunCommand(ctx, "git status")
			foundation.RunCommand(ctx, "git add --all")
			foundation.RunCommandWithArgs(ctx, "git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'%v v%v'", params.Chart, params.Version)})
			foundation.RunCommand(ctx, "git push origin master")
		}

	case "purge":
		log.Info().Msgf("Purging pre-release version for chart %v with versions '%v-.+'...", params.Chart, params.Version)

		foundation.RunCommand(ctx, "mkdir -p %v/%v", params.RepositoryDirectory, params.RepositoryChartsSubdirectory)
		err = os.Chdir(params.RepositoryDirectory)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed changing directory to %v", params.RepositoryDirectory)
		}

		filesGlob := fmt.Sprintf("%v/%v-%v-*.tgz", params.RepositoryChartsSubdirectory, params.Chart, params.Version)
		log.Info().Msgf("glob: %v", filesGlob)
		files, err := filepath.Glob(filesGlob)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed globbing %v", filesGlob)
		}
		if len(files) > 0 {
			foundation.RunCommand(ctx, "rm -f %v", strings.Join(files, " "))

			log.Info().Msgf("Generating/updating index file for repository %v...", params.RepositoryURL)
			foundation.RunCommand(ctx, "helm repo index --url %v .", params.RepositoryURL)

			log.Info().Msg("Pushing changes to repository...")
			foundation.RunCommandWithArgs(ctx, "git", []string{"config", "--global", "user.email", "'bot@estafette.io'"})
			foundation.RunCommandWithArgs(ctx, "git", []string{"config", "--global", "user.name", "'estafette-bot'"})
			foundation.RunCommand(ctx, "git add --all")
			foundation.RunCommandWithArgs(ctx, "git", []string{"commit", "--allow-empty", "-m", fmt.Sprintf("'purged %v v%v-.+'", params.Chart, params.Version)})
			foundation.RunCommand(ctx, "git push origin master")

		} else {
			log.Info().Msg("Found 0 files to purge")
		}

	case "diff", "install":
		log.Info().Msgf("Installing chart %v with app version %v and version %v...", params.Chart, params.AppVersion, params.Version)

		initKubectl(ctx, params)

		overrideValuesFilesParameter := ""
		if params.Values != "" {
			log.Info().Msg("Writing values to override.yaml...")
			err = ioutil.WriteFile("override.yaml", []byte(params.Values), 0644)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed writing override.yaml")
			}
			overrideValuesFilesParameter = "-f override.yaml"
			foundation.RunCommand(ctx, "cat override.yaml")
		}

		filename := fmt.Sprintf("%v-%v.tgz", params.Chart, params.Version)
		if !foundation.FileExists(filename) {
			log.Info().Msgf("No helm package present, retrieving helm chart %v version %v from %v...", params.Chart, params.Version, params.RepositoryURL)
			foundation.RunCommand(ctx, "helm fetch %v --version %v --repo %v", params.Chart, params.Version, params.RepositoryURL)
		}

		log.Info().Msg("Showing template to be installed...")
		foundation.RunCommand(ctx, "helm diff upgrade %v %v %v --namespace %v --allow-unreleased", params.ReleaseName, filename, overrideValuesFilesParameter, params.Namespace)

		if params.Action == "install" {
			log.Printf("\nInstalling chart and waiting for %v for it to be ready...\n", params.Timeout)
			forceArgument := ""
			if params.Force {
				forceArgument = "--force"
			}
			err = foundation.RunCommandExtended(ctx, "helm upgrade --install %v %v %v --namespace %v --history-max 1 --cleanup-on-fail --atomic --timeout %v %v", params.ReleaseName, filename, overrideValuesFilesParameter, params.Namespace, params.Timeout, forceArgument)
			if err != nil {
				log.Printf("Installation failed, showing logs...")
				foundation.RunCommand(ctx, "kubectl get all -n %v", params.Namespace)
				_ = foundation.RunCommandExtended(ctx, "kubectl logs -l app.kubernetes.io/instance=%v -n %v --all-containers=true", params.ReleaseName, params.Namespace)
				os.Exit(1)
			}

			log.Info().Msg("Showing logs for container...")
			if params.FollowLogs {
				_ = foundation.RunCommandExtended(ctx, "kubectl logs -l app.kubernetes.io/instance=%v -n %v --all-containers=true --pod-running-timeout=60s --follow=true", params.ReleaseName, params.Namespace)
			} else {
				_ = foundation.RunCommandExtended(ctx, "kubectl logs -l app.kubernetes.io/instance=%v -n %v --all-containers=true --pod-running-timeout=60s", params.ReleaseName, params.Namespace)
			}
		}

	case "uninstall":
		log.Info().Msgf("Uninstalling chart %v...", params.Chart)

		initKubectl(ctx, params)

		err = foundation.RunCommandExtended(ctx, "helm uninstall %v --namespace %v --timeout %v", params.ReleaseName, params.Namespace, params.Timeout)

	default:
		log.Fatal().Msgf("Action '%v' is not supported; please use action parameter value 'lint', 'package', 'test', 'publish', 'diff', 'install' or 'purge'", params.Action)
	}
}

func initCredential(ctx context.Context, params params) *GKECredentials {
	if *credentialsJSON == "" {
		log.Fatal().Msg("Credentials of type kubernetes-engine are not injected; configure this extension as trusted and inject credentials of type kubernetes-engine")
	}

	log.Info().Msg("Unmarshalling injected credentials...")
	var credentials []GKECredentials
	err := json.Unmarshal([]byte(*credentialsJSON), &credentials)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed unmarshalling injected credentials")
	}

	log.Info().Msgf("Checking if credential %v exists...", params.Credentials)
	credential := GetCredentialsByName(credentials, params.Credentials)
	if credential == nil {
		log.Fatal().Msgf("Credential with name %v does not exist.", params.Credentials)
	}

	log.Info().Msgf("Storing gcp credential %v on disk...", params.Credentials)
	err = ioutil.WriteFile("/key-file.json", []byte(credential.AdditionalProperties.ServiceAccountKeyfile), 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed writing service account keyfile")
	}

	log.Info().Msg("Retrieving service account email from credentials...")
	var keyFileMap map[string]interface{}
	err = json.Unmarshal([]byte(credential.AdditionalProperties.ServiceAccountKeyfile), &keyFileMap)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed unmarshalling service account keyfile")
	}

	var saClientEmail string
	if saClientEmailIntfc, ok := keyFileMap["client_email"]; !ok {
		log.Fatal().Msg("Field client_email missing from service account keyfile")
	} else {
		if t, aok := saClientEmailIntfc.(string); !aok {
			log.Fatal().Msg("Field client_email not of type string")
		} else {
			saClientEmail = t
		}
	}

	log.Info().Msg("Authenticating to google cloud")
	foundation.RunCommandWithArgs(ctx, "gcloud", []string{"auth", "activate-service-account", saClientEmail, "--key-file", "/key-file.json"})

	log.Info().Msgf("Setting gcloud account to %v", saClientEmail)
	foundation.RunCommandWithArgs(ctx, "gcloud", []string{"config", "set", "account", saClientEmail})

	return credential
}

func initKubectl(ctx context.Context, params params) {

	credential := initCredential(ctx, params)

	log.Info().Msg("Setting gcloud project")
	foundation.RunCommandWithArgs(ctx, "gcloud", []string{"config", "set", "project", credential.AdditionalProperties.Project})

	log.Info().Msgf("Getting gke credentials for cluster %v", credential.AdditionalProperties.Cluster)
	clustersGetCredentialsArsgs := []string{"container", "clusters", "get-credentials", credential.AdditionalProperties.Cluster}
	if credential.AdditionalProperties.Zone != "" {
		clustersGetCredentialsArsgs = append(clustersGetCredentialsArsgs, "--zone", credential.AdditionalProperties.Zone)
	} else if credential.AdditionalProperties.Region != "" {
		clustersGetCredentialsArsgs = append(clustersGetCredentialsArsgs, "--region", credential.AdditionalProperties.Region)
	} else {
		log.Fatal().Msg("Credentials have no zone or region; at least one of them has to be defined")
	}
	foundation.RunCommandWithArgs(ctx, "gcloud", clustersGetCredentialsArsgs)
}
