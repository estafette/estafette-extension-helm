# estafette-extension-helm
This extension helps with linting, packaging, testing and adding Helm charts to repositories

## Parameters

| Parameter         | Type     | Values |
| ----------------- | -------- | ------ |
| `chart`           | string   | The name of the chart and subdirectory where the chart is stored; defaults to `$ESTAFETTE_LABEL_APP` or `$ESTAFETTE_GIT_NAME` in that order |
| `prerelease`      | bool     | Can be set to create a pre-release package; defaults to false                                                                               |
| `appVersion`      | string   | Can be used to override the app version; defaults to `ESTAFETTE_BUILD_VERSION`                                                              |
| `version`         | string   | Can be used to override the package version; defauls to `${ESTAFETTE_BUILD_VERSION_MAJOR}.${ESTAFETTE_BUILD_VERSION_MINOR}.0` for release packages and `${ESTAFETTE_BUILD_VERSION_MAJOR}.${ESTAFETTE_BUILD_VERSION_MINOR}.0-pre-${ESTAFETTE_BUILD_VERSION_PATCH}` for pre-release packages |
| `kindHost`        | string   | The service container name running the [bsycorp/kind](https://hub.docker.com/r/bsycorp/kind) container to run tests against                 |
| `timeout`         | int      | The time in seconds to wait for install during the `test` action to finish; defaults to 200 seconds                                         |
| `repoDir`         | string   | The directory into which the chart repository is cloned; defaults to `helm-charts`                                                          |
| `chartsSubdir`    | string   | The subdirectory of the chart repository into which the tgz files are copied; defaults to `charts`                                          |
| `repoUrl`         | string   | The full url towards the helm repository, to be used to generate the `index.yaml` file; defaults to `https://helm.estafette.io/`            |
| `purgePrerelease` | bool     | Can be set to purge pre-release packages during the `publish` action; this will delete all pre-release packages for this chart from the chart repository |
| `values`          | []string | Array of values to pass to the install command during the `test` action in order to set required values                                     |

## Usage

In order to use this extension in your `.estafette.yaml` manifest for the various supported actions use the following snippets:

### Linting

```yaml
  lint-helm-chart:
    image: extensions/helm:stable
    action: lint
    prerelease: true
```

### Packaging

```yaml
  package-helm-chart:
    image: extensions/helm:stable
    action: package
    prerelease: true
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
    prerelease: true
    values:
    - secret.cloudflareApiEmail=bot@estafette.io
    - secret.cloudflareApiKey=abc
```

### Publishing

In order to publish to a git repository you first need to clone that git repository, then you can run

```yaml
  clone-charts-repo:
    image: extensions/git-clone:dev
    repo: helm-charts
    branch: master
    subdir: helm-charts

  publish-helm-chart:
    image: extensions/helm:stable
    action: publish
    prerelease: true
    repoDir: helm-charts
    chartsSubdir: charts
    repoUrl: https://helm.estafette.io/
```

### Release

When you want to release a package as a regular version you can drop the `prerelease: true` parameter and run actions `package` and `publish` like below:

```yaml
releases:
  release:
    clone: true
    stages:
      package-helm-chart:
        image: extensions/helm:stable
        action: package

      clone-charts-repo:
        image: extensions/git-clone:dev
        repo: helm-charts
        branch: master

      publish-helm-chart:
        image: extensions/helm:stable
        action: publish
```