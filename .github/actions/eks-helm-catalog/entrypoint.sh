#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks
kubectl create namespace catalog
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install catalog svc-cat/catalog --namespace catalog
