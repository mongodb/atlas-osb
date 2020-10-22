#!/bin/bash
set -e
source ".github/base-dockerfile/helpers/params.sh"
source ".github/base-dockerfile/helpers/tmp-helper.sh"

make_creds
echo "$INPUT_KUBE_CONFIG_DATA" >> kubeconfig
export KUBECONFIG="./kubeconfig"

kubectl create namespace catalog
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install catalog svc-cat/catalog --namespace catalog
