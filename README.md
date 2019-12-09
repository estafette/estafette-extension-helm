# extensions/helm

This extension helps with linting, packaging, testing and adding Helm charts to repositories

## Parameters

| Parameter             | Type   | Values                                                                                                                                              |
| --------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `action`              | string | Determines the action taken by the extension; valid options are `lint`, `package`, `test`, `publish`, `diff`, `install` or `purge`                  |
| `appVersion`          | string | Can be used to override the app version; defaults to `$ESTAFETTE_BUILD_VERSION`                                                                     |
| `chart`               | string | The name of the chart and subdirectory where the chart is stored; defaults to `$ESTAFETTE_LABEL_APP` or `$ESTAFETTE_GIT_NAME` in that order         |
| `credentials`         | string | To set a specific set of type `kubernetes-engine` credentials when using action `install`; defaults to the release target name prefixed with `gke-` |
| `force`               | bool   | Allow a force installation for action `install`                                                                                                     |
| `helmSubdir`          | string | The subdirectory in this repository where helm charts are stores; defaults to `helm`                                                                |
| `kindHost`            | string | The service container name running the [bsycorp/kind](https://hub.docker.com/r/bsycorp/kind) container to run tests against                         |
| `namespace`           | string | The namespace to deploy to when using action `install`                                                                                              |
| `releaseName`         | string | Name for the Helm release created with action `install`; defaults to the `chart` name                                                               |
| `repoDir`             | string | The directory into which the chart repository is cloned; defaults to `helm-charts`                                                                  |
| `repoChartsSubdir`    | string | The subdirectory of the chart repository into which the tgz files are copied; defaults to `charts`                                                  |
| `repoUrl`             | string | The full url towards the helm repository, to be used to generate the `index.yaml` file; defaults to `https://helm.estafette.io/`                    |
| `timeout`             | int    | The time in seconds to wait for install during the `test` action to finish; defaults to 200 seconds                                                 |
| `values`              | string | Contents of a values.yaml files to use with the install command during the `test` action in order to set required values                            |
| `version`             | string | Can be used to override the package version; defauls to `$ESTAFETTE_BUILD_VERSION`                                                                  |

## Usage

In order to use this extension in your `.estafette.yaml` manifest for the various supported actions use the following snippets:

### Linting

```yaml
  lint-helm-chart:
    image: extensions/helm:stable
    action: lint
```

### Packaging

```yaml
  package-helm-chart:
    image: extensions/helm:stable
    action: package
```

### Testing

Testing depends on Estafette's service containers to provide a Kubernetes environment inside a container running in the background.

```yaml
  test-helm-chart:
    services:
    - name: kubernetes
      image: bsycorp/kind:latest-1.12
      readiness:
        path: /kubernetes-ready
        port: 10080
    image: extensions/helm:stable
    action: test
    values: |-
      secret:
        cloudflareApiEmail=bot@estafette.io
        cloudflareApiKey=abc
```

Note: For the above to work make sure image `bsycorp/kind` is configured as _trusted image_ with `runPrivileged: true`.

### Publishing

In order to publish to a git repository you first need to clone that git repository and then run the `publish` action as follows:

```yaml
  clone-charts-repo:
    image: extensions/git-clone:stable
    repo: helm-charts
    branch: master
    subdir: helm-charts

  publish-helm-chart:
    image: extensions/helm:stable
    action: publish
    repoDir: helm-charts
    chartsSubdir: charts
    repoUrl: https://helm.estafette.io/
```

### Release version

If you run the package and publish steps as shown above it will take the version coming from Estafette CI. If you're on a release branch it will drop the label from the version number, automatically leading to a release package.

To ensure this only happens on the branch for your release version use the following snippet for defining your build version:

```yaml
version:
  semver:
    major: 1
    minor: 2
    patch: 1
    labelTemplate: 'beta-{{auto}}'
    releaseBranch: 1.2.1
```

This moves the autoincrementing build number from the `patch` field - which is default - to the label and with the `releaseBranch` set to the version you intend to release it will not build a release package until you create and push that branch.

### Purge

When releasing your final version you might want to purge the pre-release versions leading up to that moment. You can do this with `action: purge`, for example in a release target. You'll have to clone the Helm chart repository first:

```yaml
releases:
  release-helm-chart:
    stages:
      clone-charts-repo:
        image: extensions/git-clone:stable
        repo: helm-charts
        branch: master

      purge-prerelease-helm-charts:
        image: extensions/helm:stable
        action: purge

      create-github-release:
        image: extensions/github-release:stable
```

Notice at the end it creates a Github release, for which it expects a milestone to be present with the title equal to `${ESTAFETTE_BUILD_VERSION}`.

### Install

The Helm extension is configured as a _trusted image_ and gets credentials of type _kubernetes-engine_ injected; by default the release target name gets prefixed with `gke-` to select the credentials, or it can be set explicitly with the `credentials` parameter.

```yaml
releases:
  development:
    stages:
      install:
        image: extensions/helm:stable
        action: install
        namespace: mynamespace
        repoUrl: https://helm.estafette.io
        values: |-
          secret:
            cloudflareApiEmail=bot@estafette.io
            cloudflareApiKey=abc
```

The install will try to use the package from the repository it's previously been pushed to. You can also use the local chart in the following way in order to install charts that haven't been published:

```yaml
releases:
  clone: true
  development:
    stages:
      package-helm-chart:
        image: extensions/helm:stable
        action: package

      install:
        image: extensions/helm:stable
        action: install
        namespace: mynamespace
```

### Diff

The _diff_ action does everything the _install_ action does except for actually applying the changes, with the following snippet.

```yaml
releases:
  development:
    stages:
      diff:
        image: extensions/helm:stable
        action: diff
        namespace: mynamespace
        repoUrl: https://helm.estafette.io
        values: |-
          secret:
            cloudflareApiEmail=bot@estafette.io
            cloudflareApiKey=abc
```
