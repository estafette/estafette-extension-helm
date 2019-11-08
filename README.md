# extensions/helm

This extension helps with linting, packaging, testing and adding Helm charts to repositories

## Parameters

| Parameter         | Type     | Values |
| ----------------- | -------- | ------ |
| `chart`           | string   | The name of the chart and subdirectory where the chart is stored; defaults to `$ESTAFETTE_LABEL_APP` or `$ESTAFETTE_GIT_NAME` in that order |
| `appVersion`      | string   | Can be used to override the app version; defaults to `$ESTAFETTE_BUILD_VERSION`                                                             |
| `version`         | string   | Can be used to override the package version; defauls to `$ESTAFETTE_BUILD_VERSION`                                                          |
| `kindHost`        | string   | The service container name running the [bsycorp/kind](https://hub.docker.com/r/bsycorp/kind) container to run tests against                 |
| `timeout`         | int      | The time in seconds to wait for install during the `test` action to finish; defaults to 200 seconds                                         |
| `repoDir`         | string   | The directory into which the chart repository is cloned; defaults to `helm-charts`                                                          |
| `chartsSubdir`    | string   | The subdirectory of the chart repository into which the tgz files are copied; defaults to `charts`                                          |
| `repoUrl`         | string   | The full url towards the helm repository, to be used to generate the `index.yaml` file; defaults to `https://helm.estafette.io/`            |
| `values`          | string   | Contents of a values.yaml files to use with the install command during the `test` action in order to set required values                    |

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
      ports:
      - port: 8443
      - port: 10080
        readiness:
          path: /kubernetes-ready
          timeoutSeconds: 180
    image: extensions/helm:stable
    action: test
    values: |-
      secret:
        cloudflareApiEmail=bot@estafette.io
        cloudflareApiKey=abc
```

### Publishing

In order to publish to a git repository you first need to clone that git repository and then run the `publish` action as follows:

```yaml
  clone-charts-repo:
    image: extensions/git-clone:dev
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

When releasing your final version you might want to purge the pre-release versions leading up to that moment. You can do sith with `action: purge`, for example in a release target. You'll have to clone the Helm chart repository first:

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

Notice at the end it creates a Github release, for which it expects a milestone to be present with the title equal to `$ESTAFETTE_BUILD_VERSION`.