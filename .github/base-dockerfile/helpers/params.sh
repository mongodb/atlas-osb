#shellcheck shell=bash disable=SC2034

#make `act` -e works
if [[ -n $BRANCH ]]
then
    ref=$BRANCH
else
    ref=$GITHUB_REF
fi
branch_name=$(echo "$ref" | awk -F'/' '{print $3}')
branch_name=${branch_name:0:10} #service name max length is 23 symbols minus prefixes (5) postfix (8)
# instance_name is used for Atlas project & cluster name, but cluster names need to follow
# The name can only contain ASCII letters, numbers, and hyphens.
# Here we only catch '.'dot's for release builds.
branch_name=$(echo "${branch_name}" | tr "." "-")
commit_id=$(git rev-parse --short HEAD)
postfix=$branch_name-$commit_id

#arguments for actions
ORG_PREFIX="atlas-test-"
if [[ $TEST_TYPE == "go" ]]; then #override for go-test, TODO: delete after leaving only one type of tests
    ORG_PREFIX="atlas-gt-"
    postfix="$postfix-2"
    s_post="-go"
fi
ORG_NAME="$ORG_PREFIX$branch_name"
SPACE_NAME=$commit_id$s_post
BROKER=atlas-osb-$postfix
BROKER_APP=atlas-osb-app-$postfix
CREDHUB=credhub-$postfix
TEST_SIMPLE_APP=simple-app-$postfix
TEST_SPRING_APP=music-$postfix
SERVICE_ATLAS=inst-$postfix
SERVICE_ATLAS_RENAME=$SERVICE_ATLAS-rnm
BROKER_OSB_SERVICE_NAME="atlas"

#k8s default demo names
K_NAMESPACE="atlas-$postfix"
K_BROKER="aosb-$commit_id"
K_SERVICE="aosbs-$postfix"
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