package main

import "fmt"

type params struct {
	Action                       string `json:"action,omitempty" yaml:"action,omitempty"`
	AppVersion                   string `json:"appVersion,omitempty" yaml:"appVersion,omitempty"`
	Chart                        string `json:"chart,omitempty" yaml:"chart,omitempty"`
	Credentials                  string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	FollowLogs                   bool   `json:"followLogs,omitempty" yaml:"followLogs,omitempty"`
	Force                        bool   `json:"force,omitempty" yaml:"force,omitempty"`
	HelmSubdirectory             string `json:"helmSubdir,omitempty" yaml:"helmSubdir,omitempty"`
	KindHost                     string `json:"kindHost,omitempty" yaml:"kindHost,omitempty"`
	LabelSelectorOverride        string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	Namespace                    string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ReleaseName                  string `json:"release,omitempty" yaml:"release,omitempty"`
	RepositoryDirectory          string `json:"repoDir,omitempty" yaml:"repoDir,omitempty"`
	RepositoryChartsSubdirectory string `json:"repoChartsSubdir,omitempty" yaml:"repoChartsSubdir,omitempty"`
	RepositoryURL                string `json:"repoUrl,omitempty" yaml:"repoUrl,omitempty"`
	Bucket                       string `json:"bucket,omitempty" yaml:"bucket,omitempty"`
	Timeout                      string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
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

	if p.Timeout == "" {
		p.Timeout = "120s"
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

	// default credentials to release name prefixed with gke if no override in stage params
	if p.Credentials == "" && releaseTargetName != "" {
		p.Credentials = fmt.Sprintf("gke-%v", releaseTargetName)
	}
}

type requirements struct {
	Dependencies []dependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

type dependency struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Version    string `json:"version,omitempty" yaml:"version,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Alias      string `json:"alias,omitempty" yaml:"alias,omitempty"`
}
