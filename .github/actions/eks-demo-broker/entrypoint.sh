#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks

set -e
echo "leori/atlas-osb:${branch_name}"
helm install "${K_BROKER}" \
    --set namespace="${K_NAMESPACE}" \
    --set image="leori/atlas-osb:${branch_name}" \
    --set atlas.orgId="${INPUT_ATLAS_ORG_ID}" \
    --set atlas.publicKey="${INPUT_ATLAS_PUBLIC_KEY}" \
    --set atlas.privateKey="${INPUT_ATLAS_PRIVATE_KEY}" \
    --set broker.auth.username="${K_DEFAULT_USER}" \
    --set broker.auth.password="${K_DEFAULT_PASS}" \
    samples/helm/broker/ --namespace "${K_NAMESPACE}" --wait --timeout 10m --create-namespace

kubectl get all -n "${K_NAMESPACE}"
