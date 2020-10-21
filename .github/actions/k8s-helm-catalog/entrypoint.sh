#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

echo "$INPUT_KUBE_CONFIG_DATA" >> kubeconfig
export KUBECONFIG="./kubeconfig"

kubectl create namespace catalog
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install catalog svc-cat/catalog --namespace catalog
