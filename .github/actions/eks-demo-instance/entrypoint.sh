#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks

helm install "${K_SERVICE}" samples/helm/sample-service/ \
    --set broker.auth.username="${K_DEFAULT_USER}" \
    --set broker.auth.password="${K_DEFAULT_PASS}" \
    --namespace "${K_NAMESPACE}" --wait --timeout 60m

kubectl get all -n "${K_NAMESPACE}"
