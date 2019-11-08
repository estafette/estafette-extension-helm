#!/bin/bash
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

appversion="${ESTAFETTE_BUILD_VERSION}"
if [ "${ESTAFETTE_EXTENSION_APP_VERSION}" != "" ]; then
    appversion="${ESTAFETTE_EXTENSION_APP_VERSION}"
fi

version="${ESTAFETTE_BUILD_VERSION}"
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

values=()
if [ "${ESTAFETTE_EXTENSION_VALUES}" != "" ]; then
    IFS=',' read -r -a values <<< "${ESTAFETTE_EXTENSION_VALUES}"
fi
setvalues=""
for v in "${values[@]}"; do
    setvalues+="--set $v "
done

case $ESTAFETTE_EXTENSION_ACTION in
lint)
    echo "Linting chart $chart..."
    helm lint $chart
    ;;
package)
    echo "Packaging chart $chart with app version $appversion and version $version..."
    helm package --save=false --app-version $appversion --version $version $chart
    ;;
test)
    echo "Testing chart $chart with app version $appversion and version $version on kind host $kindhost..."

    printf "\nWaiting for kind host to be ready...\n"
    while true; do
        wget -T 1 -c http://${kindhost}:10080/kubernetes-ready && break
    done

    printf "\nPreparing kind host for using Helm...\n"
    wget -q -O - http://${kindhost}:10080/config | sed -e "s/localhost/${kindhost}/" > ~/.kube/config
    kubectl -n kube-system create serviceaccount tiller
    kubectl create clusterrolebinding tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
    helm init --service-account tiller --wait

    if [ "${setvalues}" != "" ]; then
        printf "Using following arguments for setting values:"
        echo "'${setvalues}'"
    fi

    printf "\nShowing template to be installed...\n"
    helm template --name $chart $chart-$version.tgz $setvalues
    
    printf "\nInstalling chart and waiting for ${timeout}s for it to be ready...\n"
    helm upgrade --install $chart $chart-$version.tgz $setvalues --wait --timeout $timeout || (kubectl logs -l app.kubernetes.io/name=${chart},app.kubernetes.io/instance=${chart} && exit 1)

    printf "\nShowing logs for container...\n"
    kubectl logs -l app.kubernetes.io/name=${chart},app.kubernetes.io/instance=${chart}
    ;;
publish)
    echo "Publishing chart $chart with app version $appversion and version $version..."

    mkdir -p ${repodir}/${chartssubdir}

    cp *.tgz ${repodir}/${chartssubdir}
    cd ${repodir}

    printf "\nGenerating/updating index file for repository $repourl...\n"
    helm repo index --url $repourl .

    printf "\nPushing changes to repository...\n"
    git config --global user.email "bot@estafette.io"
    git config --global user.name "Estafette bot"
    git add --all
    git commit --allow-empty -m "${chart} v${version}"
    git push origin master
    ;;
purge)
    echo "Purging pre-release version for chart $chart with versions '$version-.+'..."

    mkdir -p ${repodir}/${chartssubdir}

    cd ${repodir}

    rm -f ${repodir}/${chartssubdir}/${chart}-${version}-*.tgz

    printf "\nGenerating/updating index file for repository $repourl...\n"
    helm repo index --url $repourl .

    printf "\nPushing changes to repository...\n"
    git config --global user.email "bot@estafette.io"
    git config --global user.name "Estafette bot"
    git add --all
    git commit --allow-empty -m "purged ${chart} v${version}-.+"
    git push origin master
    ;;
*)
    echo "Action '$ESTAFETTE_EXTENSION_ACTION' is not supported; please use action parameter value 'lint','package','test' or 'publish'"
    exit 1
    ;;
esac