#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
set -e

service="${INPUT_SERVICE}"
app_name="test-app-${service}"
if [[ -z "${service}" ]]; then
    service="${K_DEFAULT_SERVICE}"
    app_name="${K_TEST_APP}"
fi
namespace="${INPUT_NAMESPACE}"
if [[ -z "${namespace}" ]]; then
    namespace="${K_NAMESPACE}"
fi
echo "Instance ${service} in ${namespace}"

aws eks --region us-east-2 update-kubeconfig --name atlas-osb-eks

helm install "${app_name}" samples/helm/test-app/ \
    --set service.name="${service}" \
    --namespace "${namespace}" --wait --timeout 60m

kubectl get all -n "${namespace}"

#fast check
app_url=$(kubectl get services -n ${namespace} | awk '/'"${app_name}"'/{print $4":"$5}' | awk -F':' '{print $1":"$2}')
data='{"_class":"org.cloudfoundry.samples.music.domain.Album", "artist": "Tenno", "title": "Journey", "releaseYear": "2019", "genre": "chillhop" }'
curl -H "Content-Type: application/json" -X PUT \
    -d  "${data}" "${app_url}/albums"
result=$(curl -X GET "${app_url}/albums" -s | awk '/Tenno/{print "true"}')
echo "====================================================================="
echo "GET result ${result}"

#summary
echo "====================================================================="
echo "namespace: ${namespace}"
echo "test-app: http://${app_url}"
