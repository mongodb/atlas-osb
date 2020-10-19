#!/bin/bash
# Requires having service catalog installed https://kubernetes.io/docs/tasks/service-catalog/install-service-catalog-using-helm/
# Please run `act -j eksdemo-catalog` (eks-deploy-catalog.yml) for installing service catalog

source ".github/base-dockerfile/helpers/params.sh"

helm version
aws --version
aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks
kubectl version

broker_user="admin"
broker_pass="admin"

set -e

helm install "${K_BROKER}" \
    --set atlas.orgId="${INPUT_ATLAS_ORG_ID}" \
    --set atlas.publicKey="${INPUT_ATLAS_PUBLIC_KEY}" \
    --set atlas.privateKey="${INPUT_ATLAS_PRIVATE_KEY}" \
    --set broker.auth.username="${broker_user}" \
    --set broker.auth.password="${broker_pass}" \
    samples/helm/broker/ --namespace "${K_NAMESPACE}" --wait --timeout 10m --create-namespace

helm install "${K_SERVICE}" samples/helm/sample-service/ \
    --set broker.auth.username="${broker_user}" \
    --set broker.auth.password="${broker_pass}" \
    --namespace "${K_NAMESPACE}" --wait --timeout 60m

helm install "${K_TEST_APP}" samples/helm/test-app/ \
    --set service.name="${K_SERVICE}" \
    --namespace "${K_NAMESPACE}" --wait --timeout 60m

kubectl get all -n "${K_NAMESPACE}"

#fast check
app_url=$(kubectl get services -n atlas-k8s-sample-8d390a8 | awk '/LoadBalancer/{print $4":"$5}' | awk -F':' '{print $1":"$2}')
data='{"_class":"org.cloudfoundry.samples.music.domain.Album", "artist": "Tenno", "title": "Journey", "releaseYear": "2019", "genre": "chillhop" }'
curl -H "Content-Type: application/json" -X PUT \
    -d  "${data}" "${app_url}/albums"
result=$(curl -X GET "${app_url}/albums" -s | awk '/Tenno/{print "true"}')
echo "GET result ${result}"

#summary
echo "====================================================================="
echo "namespace: ${K_NAMESPACE}"
echo "test-app: http://${app_url}"
