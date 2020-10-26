#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks

helm install "${K_TEST_APP}" samples/helm/test-app/ \
    --set service.name="${K_SERVICE}" \
    --namespace "${K_NAMESPACE}" --wait --timeout 60m

kubectl get all -n "${K_NAMESPACE}"

#fast check
sleep 10s
app_url=$(kubectl get services -n "${K_NAMESPACE}" | awk '/'"${K_TEST_APP}"'/{print $4":"$5}' | awk -F':' '{print $1":"$2}')
data='{"_class":"org.cloudfoundry.samples.music.domain.Album", "artist": "Tenno", "title": "Journey", "releaseYear": "2019", "genre": "chillhop" }'
curl -H "Content-Type: application/json" -X PUT \
    -d  "${data}" "${app_url}/albums"
result=$(curl -X GET "${app_url}/albums" -s | awk '/Tenno/{print "true"}')
echo "====================================================================="
echo "GET result ${result}"

#summary
echo "====================================================================="
echo "namespace: ${K_NAMESPACE}"
echo "test-app: http://${app_url}"
