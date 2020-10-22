#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
source ".github/base-dockerfile/helpers/tmp-helper.sh"

make_creds
echo "$INPUT_KUBE_CONFIG_DATA" >> kubeconfig
export KUBECONFIG="./kubeconfig"

helm install "${K_TEST_APP}" samples/helm/test-app/ \
    --set service.name="${K_SERVICE}" \
    --namespace "${K_NAMESPACE}" --wait --timeout 60m

kubectl get all -n "${K_NAMESPACE}"

#summary
app_url=$(kubectl get services -n "${K_NAMESPACE}" | awk '/'"${K_TEST_APP}"'/{print $4":"$5}' | awk -F':' '{print $1":"$2}')
echo "====================================================================="
echo "namespace: ${K_NAMESPACE}"
echo "test-app: http://${app_url}"

#EKS
data='{"_class":"org.cloudfoundry.samples.music.domain.Album", "artist": "Tenno", "title": "Journey", "releaseYear": "2019", "genre": "chillhop" }'
curl -H "Content-Type: application/json" -X PUT \
    -d  "${data}" "${app_url}/albums"
result=$(curl -X GET "${app_url}/albums" -s | awk '/Tenno/{print "true"}')
echo "====================================================================="
echo "${result}"
