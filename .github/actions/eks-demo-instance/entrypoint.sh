#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

service="${INPUT_SERVICE}"
if [[ -z "${service}" ]]; then
    service="${K_DEFAULT_SERVICE}"
fi
namespace="${INPUT_NAMESPACE}"
if [[ -z "${namespace}" ]]; then
    namespace="${K_NAMESPACE}"
fi
echo "Instance ${service} in ${namespace}"

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks

helm install "${service}" samples/helm/sample-service/ \
    --set broker.auth.username="${K_DEFAULT_USER}" \
    --set broker.auth.password="${K_DEFAULT_PASS}" \
    --namespace "${namespace}" --wait --timeout 60m
