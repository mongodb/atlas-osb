#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

helm version
aws --version
aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks
kubectl version

#delete instances first
output=$(kubectl get all -n atlas-man4)
echo "${output}" | awk '/servicebinding.servicecatalog.k8s.io/{system("kubectl delete " $1 " -n atlas-man4")}'
echo "${output}" | awk '/ServiceClass\/atlas/{system("kubectl delete " $1 " -n atlas-man4")}'

helm uninstall "${K_APP_NAME}" \
    --namespace "${K_NAMESPACE}"

helm uninstall "${K_SERVICE}" \
    --namespace "${K_NAMESPACE}"

helm uninstall "${K_BROKER}" \
    --namespace "${K_NAMESPACE}"

kubectl get all -n "${K_NAMESPACE}"
# kubectl delete namespace "${NAMESPACE}"