#!/bin/sh
set -e

echo "Starting estafette-extension-helm version ${VERSION}..."

# set chart name
chart="${ESTAFETTE_GIT_NAME}"
if [ "${ESTAFETTE_LABEL_APP}" != "" ]; then
    chart="${ESTAFETTE_LABEL_APP}"
fi
if [ "${ESTAFETTE_EXTENSION_CHART}" != "" ]; then
    chart="${ESTAFETTE_EXTENSION_CHART}"
fi

prerelease="false"
if [ "${ESTAFETTE_EXTENSION_PRERELEASE}" != "" ]; then
    prerelease="${ESTAFETTE_EXTENSION_PRERELEASE}"
fi

appversion="${ESTAFETTE_BUILD_VERSION}"
if [ "${ESTAFETTE_EXTENSION_APP_VERSION}" != "" ]; then
    appversion="${ESTAFETTE_EXTENSION_APP_VERSION}"
fi

version="${ESTAFETTE_BUILD_VERSION_MAJOR}.${ESTAFETTE_BUILD_VERSION_MINOR}.0"
if [ "$prerelease" == "true" ]; then
    version="${ESTAFETTE_BUILD_VERSION_MAJOR}.${ESTAFETTE_BUILD_VERSION_MINOR}.0-pre-${ESTAFETTE_BUILD_VERSION_PATCH}"
fi
if [ "${ESTAFETTE_EXTENSION_VERSION}" != "" ]; then
    version="${ESTAFETTE_EXTENSION_VERSION}"
fi

kindhost="kubernetes"
if [ "${ESTAFETTE_EXTENSION_KIND_HOST}" != "" ]; then
    kindhost="${ESTAFETTE_EXTENSION_KIND_HOST}"
fi

timeout="120"
if [ "${ESTAFETTE_EXTENSION_TIMEOUT}" != "" ]; then
    timeout="${ESTAFETTE_EXTENSION_TIMEOUT}"
fi

repodir="helm-charts"
if [ "${ESTAFETTE_EXTENSION_REPO_DIR}" != "" ]; then
    repodir="${ESTAFETTE_EXTENSION_REPO_DIR}"
fi

chartssubdir="charts"
if [ "${ESTAFETTE_EXTENSION_CHARTS_SUBDIR}" != "" ]; then
    chartssubdir="${ESTAFETTE_EXTENSION_CHARTS_SUBDIR}"
fi

repourl="https://helm.estafette.io/"
if [ "${ESTAFETTE_EXTENSION_REPO_URL}" != "" ]; then
    repourl="${ESTAFETTE_EXTENSION_REPO_URL}"
fi

purgeprerelease="false"
if [ "${ESTAFETTE_EXTENSION_PURGE_PRERELEASE}" != "" ]; then
    purgeprerelease="${ESTAFETTE_EXTENSION_PURGE_PRERELEASE}"
fi

case $ESTAFETTE_EXTENSION_ACTION in
lint)
    echo "Linting chart $chart..."
    helm lint $chart
    break
    ;;
package)
    echo "Packaging chart $chart with app version $appversion and version $version..."
    helm package --save=false --app-version $appversion --version $version $chart
    break
    ;;
test)
    echo "Testing chart $chart with app version $appversion and version $version on kind host $kindhost..."

    echo "Waiting for kind host to be ready..."
    while true; do
        wget -T 1 -c http://${kindhost}:10080/kubernetes-ready && break
    done

    echo "Preparing kind host for using Helm..."
    wget -q -O - http://${kindhost}:10080/config | sed -e "s/localhost/${kindhost}/" > ~/.kube/config
    kubectl -n kube-system create serviceaccount tiller
    kubectl create clusterrolebinding tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
    helm init --service-account tiller --wait

    echo "Showing template to be installed..."
    helm template --name $chart $chart-$version.tgz
    
    echo "Installing chart and waiting for ${timeout}s for it to be ready..."
    helm upgrade --install $chart $chart-$version.tgz --wait --timeout $timeout || (kubectl logs -l app.kubernetes.io/name=${chart},app.kubernetes.io/instance=${chart} && exit 1)

    echo "Showing logs for container..."
    kubectl logs -l app.kubernetes.io/name=${chart},app.kubernetes.io/instance=${chart}

    break
    ;;
publish)
    echo "Publishing chart $chart with app version $appversion and version $version..."

    mkdir -p ${repodir}/${chartssubdir}

    if [ "$purgeprerelease" == "true" ]; then
        echo "Purging prerelease packages for chart $chart..."
        rm -f "${repodir}/${chartssubdir}/${chart}-*-pre-*.tgz"
    fi

    cp *.tgz ${repodir}/${chartssubdir}
    cd ${repodir}

    echo "Generating/updating index file for repository $repourl..."
    helm repo index --url $repourl --merge index.yaml .

    echo "Pushing changes to repository..."
    git config --global user.email "bot@estafette.io"
    git config --global user.name "Estafette bot"
    git add --all
    git commit --allow-empty -m "${version}"
    git push origin master

    break
    ;;
*)
    echo "Action '$ESTAFETTE_EXTENSION_ACTION' is not supported; please use action parameter value 'lint','package','test' or 'publish'"
    exit 1
    ;;
esac