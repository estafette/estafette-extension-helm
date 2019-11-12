package main

import "fmt"

var defaultTimeout = 120

type params struct {
	Action                       string `json:"action,omitempty" yaml:"action,omitempty"`
	AppVersion                   string `json:"appVersion,omitempty" yaml:"appVersion,omitempty"`
	Chart                        string `json:"chart,omitempty" yaml:"chart,omitempty"`
	Credentials                  string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	HelmSubdirectory             string `json:"helmSubdir,omitempty" yaml:"helmSubdir,omitempty"`
	KindHost                     string `json:"kindHost,omitempty" yaml:"kindHost,omitempty"`
	Namespace                    string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ReleaseName                  string `json:"release,omitempty" yaml:"release,omitempty"`
	RepositoryDirectory          string `json:"repoDir,omitempty" yaml:"repoDir,omitempty"`
	RepositoryChartsSubdirectory string `json:"repoChartsSubdir,omitempty" yaml:"repoChartsSubdir,omitempty"`
	RepositoryURL                string `json:"repoUrl,omitempty" yaml:"repoUrl,omitempty"`
	Tillerless                   bool   `json:"tillerless,omitempty" yaml:"tillerless,omitempty"`
	TillerlessNamespace          string `json:"tillerlessNamespace,omitempty" yaml:"tillerlessNamespace,omitempty"`
	Timeout                      *int   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Values                       string `json:"values,omitempty" yaml:"values,omitempty"`
	Version                      string `json:"version,omitempty" yaml:"version,omitempty"`
}

func (p *params) SetDefaults(gitName string, appLabel string, buildVersion string, releaseTargetName string) {

	// set chart name
	if p.Chart == "" {
		if appLabel != "" {
			p.Chart = appLabel
		} else if gitName != "" {
			p.Chart = gitName
		}
	}

	if p.AppVersion == "" && buildVersion != "" {
		p.AppVersion = buildVersion
	}

	if p.Version == "" && buildVersion != "" {
		p.Version = buildVersion
	}

	if p.KindHost == "" {
		p.KindHost = "kubernetes"
	}

	if p.Timeout == nil {
		p.Timeout = &defaultTimeout
	}

	if p.HelmSubdirectory == "" {
		p.HelmSubdirectory = "helm"
	}

	if p.RepositoryDirectory == "" {
		p.RepositoryDirectory = "helm-charts"
	}

	if p.RepositoryChartsSubdirectory == "" {
		p.RepositoryChartsSubdirectory = "charts"
	}

	if p.RepositoryURL == "" {
		p.RepositoryURL = "https://helm.estafette.io/"
	}

	if p.ReleaseName == "" {
		p.ReleaseName = p.Chart
	}

	if p.TillerlessNamespace == "" {
		p.TillerlessNamespace = "helm-extension-releases"
	}

	// default credentials to release name prefixed with gke if no override in stage params
	if p.Credentials == "" && releaseTargetName != "" {
		p.Credentials = fmt.Sprintf("gke-%v", releaseTargetName)
	}
}
