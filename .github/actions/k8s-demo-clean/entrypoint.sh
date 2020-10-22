#!/bin/bash
set -e
source ".github/base-dockerfile/helpers/params.sh"
source ".github/base-dockerfile/helpers/tmp-helper.sh"

make_creds
helm version
aws --version
echo "$INPUT_KUBE_CONFIG_DATA" >> kubeconfig
export KUBECONFIG="./kubeconfig"
kubectl version

#delete instances first
output=$(kubectl get all -n "${K_NAMESPACE}")
echo "${output}" | awk '/servicebinding.servicecatalog.k8s.io/{system("kubectl delete " $1 " -n  '"${K_NAMESPACE}"'")}'
echo "${output}" | awk '/ServiceClass\/atlas/{system("kubectl delete " $1 " -n  '"${K_NAMESPACE}"'")}'

kubectl delete namespace "${K_NAMESPACE}"
