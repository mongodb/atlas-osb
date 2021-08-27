#shellcheck shell=bash disable=SC2034

#make `act` -e works
if [[ -n $BRANCH ]]
then
    ref=$BRANCH
elif [[ -n $GITHUB_REF ]]
then
    ref=$GITHUB_REF
else
    ref="local"
fi


branch_name=$(echo "$ref" | awk -F'/' '{print $3}')
# Please, remember: service name max length is 23 symbols minus prefixes (5) postfix (8)
# # instance_name is used for Atlas project & cluster name, but cluster names need to follow
# # The name can only contain ASCII letters, numbers, and hyphens.
# # Here we only catch '.'dot's for release builds.
branch_name=$(echo "${branch_name}" | tr "." "-")
commit_id=$(git rev-parse --short HEAD)


#arguments for actions
ORG_NAME="$branch_name"
SPACE_NAME=$commit_id
BROKER=atlas-osb-$commit_id
BROKER_APP=atlas-osb-app-$commit_id
CREDHUB=credhub-$commit_id
TEST_SIMPLE_APP=simple-app-$commit_id
TEST_SPRING_APP=music-$commit_id
SERVICE_ATLAS=inst-$commit_id
SERVICE_ATLAS_RENAME=$SERVICE_ATLAS-rnm
BROKER_OSB_SERVICE_NAME="atlas"

#k8s default demo names
K_NAMESPACE="atlas-$commit_id"
K_BROKER="aosb-$commit_id"
K_SERVICE="aosbs-$commit_id"
K_TEST_APP="test-app-$commit_id"
K_DEFAULT_USER="admin"
K_DEFAULT_PASS="admin"

#override, if service/namespace presented in pipeline workflow
if [[ $INPUT_SERVICE ]]; then
    K_SERVICE=$INPUT_SERVICE
    K_TEST_APP="test-app-${K_SERVICE}"
fi
if [[ $INPUT_NAMESPACE ]]; then
    K_NAMESPACE=$INPUT_NAMESPACE
fi