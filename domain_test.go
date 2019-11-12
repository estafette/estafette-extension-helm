package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestUnmarshal(t *testing.T) {
	t.Run("ReturnsValuesIfItContainsYaml", func(t *testing.T) {

		customProperties := `
action: test
values: |-
  secret:
    letsencryptAccountJson='{}'
    letsencryptAccountKey=abc
`

		// act
		var params params
		err := yaml.Unmarshal([]byte(customProperties), &params)

		if assert.Nil(t, err) {
			assert.Equal(t, "secret:\n  letsencryptAccountJson='{}'\n  letsencryptAccountKey=abc", params.Values)
		}
	})
}

func TestSetDefaults(t *testing.T) {
	t.Run("SetsChartToAppLabelIfChartIsEmptyAndAppLabelIsNot", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Chart: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "app-label", params.Chart)
	})

	t.Run("SetsChartToGitNameIfChartIsEmptyAndAppLabelIsEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := ""
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Chart: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "git-name", params.Chart)
	})

	t.Run("KeepChartIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Chart: "mychart",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "mychart", params.Chart)
	})

	t.Run("SetsAppVersionToBuildVersionIfAppVersionIsEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			AppVersion: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "1.0.0", params.AppVersion)
	})

	t.Run("KeepsAppVersionIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			AppVersion: "2.0.0",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "2.0.0", params.AppVersion)
	})

	t.Run("SetsVersionToBuildVersionIfVersionIsEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Version: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "1.0.0", params.Version)
	})

	t.Run("KeepsVersionIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Version: "2.0.0",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "2.0.0", params.Version)
	})

	t.Run("SetsKindHostToKubernetesIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			KindHost: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "kubernetes", params.KindHost)
	})

	t.Run("KeepsKindHostIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			KindHost: "kind",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "kind", params.KindHost)
	})

	t.Run("SetsTimeoutTo120IfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Timeout: nil,
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, 120, *params.Timeout)
	})

	t.Run("KeepsTimeoutIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		timeout := 60
		params := params{
			Timeout: &timeout,
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, 60, *params.Timeout)
	})

	t.Run("SetsHelmSubdirectoryToHelmIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			HelmSubdirectory: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "helm", params.HelmSubdirectory)
	})

	t.Run("KeepsHelmSubdirectoryIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			HelmSubdirectory: "./",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "./", params.HelmSubdirectory)
	})

	t.Run("SetsRepositoryDirectoryToHelmChartsIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryDirectory: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "helm-charts", params.RepositoryDirectory)
	})

	t.Run("KeepsRepositoryDirectoryIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryDirectory: "charts-repo",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "charts-repo", params.RepositoryDirectory)
	})

	t.Run("SetsRepositoryChartsSubdirectoryToChartsIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryChartsSubdirectory: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "charts", params.RepositoryChartsSubdirectory)
	})

	t.Run("KeepsRepositoryChartsSubdirectoryIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryChartsSubdirectory: "./",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "./", params.RepositoryChartsSubdirectory)
	})

	t.Run("SetsRepositoryURLToHelmEstafetteIoIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryURL: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "https://helm.estafette.io/", params.RepositoryURL)
	})

	t.Run("KeepsRepositoryURLIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			RepositoryURL: "https://helm-beta.estafette.io/",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "https://helm-beta.estafette.io/", params.RepositoryURL)
	})

	t.Run("SetsReleaseNameToChartNameIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Chart:       "mychart",
			ReleaseName: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "mychart", params.ReleaseName)
	})

	t.Run("KeepsReleaseNameIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			Chart:       "mychart",
			ReleaseName: "myrelease",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "myrelease", params.ReleaseName)
	})

	t.Run("SetsTillerlessNamespaceToHelmExtensionReleasesIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			TillerlessNamespace: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "helm-extension-releases", params.TillerlessNamespace)
	})

	t.Run("KeepsTillerlessNamespaceIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := ""

		params := params{
			TillerlessNamespace: "helm",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "helm", params.TillerlessNamespace)
	})

	t.Run("SetsCredentialsToReleaseTargetNamePrefixedWithGKEIfEmpty", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := "development"

		params := params{
			Credentials: "",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "gke-development", params.Credentials)
	})

	t.Run("KeepsCredentialsIfSet", func(t *testing.T) {

		gitName := "git-name"
		appLabel := "app-label"
		buildVersion := "1.0.0"
		releaseTargetName := "development"

		params := params{
			Credentials: "gke-staging",
		}

		// act
		params.SetDefaults(gitName, appLabel, buildVersion, releaseTargetName)

		assert.Equal(t, "gke-staging", params.Credentials)
	})
}
