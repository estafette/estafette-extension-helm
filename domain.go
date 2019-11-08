package main

import "fmt"

var defaultTimeout = 120

type params struct {
	Action              string   `json:"action,omitempty" yaml:"action,omitempty"`
	Chart               string   `json:"chart,omitempty" yaml:"chart,omitempty"`
	AppVersion          string   `json:"appVersion,omitempty" yaml:"appVersion,omitempty"`
	Version             string   `json:"version,omitempty" yaml:"version,omitempty"`
	KindHost            string   `json:"kindHost,omitempty" yaml:"kindHost,omitempty"`
	Timeout             *int     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	RepositoryDirectory string   `json:"repoDir,omitempty" yaml:"repoDir,omitempty"`
	ChartsSubdirectory  string   `json:"chartsSubdir,omitempty" yaml:"chartsSubdir,omitempty"`
	ChartsRepositoryURL string   `json:"repoUrl,omitempty" yaml:"repoUrl,omitempty"`
	Values              []string `json:"values,omitempty" yaml:"values,omitempty"`
}

func (p *params) SetDefaults(gitName string, appLabel string, buildVersion string) {

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

	if p.RepositoryDirectory == "" {
		p.RepositoryDirectory = "helm-charts"
	}

	if p.ChartsSubdirectory == "" {
		p.ChartsSubdirectory = "charts"
	}

	if p.ChartsRepositoryURL == "" {
		p.ChartsRepositoryURL = "https://helm.estafette.io/"
	}
}

func (p *params) GetSetParameters() string {
	setParameters := ""
	if len(p.Values) > 0 {
		for _, v := range p.Values {
			setParameters += fmt.Sprintf("--set %v", v)
		}
	}

	return setParameters
}
