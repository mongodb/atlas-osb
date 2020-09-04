#shellcheck shell=bash disable=SC2034

branch_name=$(echo "$GITHUB_REF" | awk -F'/' '{print $3}')
branch_name=${branch_name:0:26} #service name max length is 50 symbols minus prefixes
# instance_name is used for Atlas project & cluster name, but cluster names need to follow
# The name can only contain ASCII letters, numbers, and hyphens.
# Here we only catch '.'dot's for release builds.
branch_name=$(echo "${branch_name}" | tr "." "-")
commit_id=$(git rev-parse --short HEAD)
postfix=$branch_name-$commit_id

#arguments for actions
ORG_PREFIX="atlas-test-"
ORG_NAME="$ORG_PREFIX$branch_name"
SPACE_NAME=$commit_id
BROKER=atlas-osb-$postfix
BROKER_APP=atlas-osb-app-$postfix
CREDHUB=credhub-$postfix
TEST_SIMPLE_APP=simple-app-$postfix
TEST_SPRING_APP=music-$postfix
SERVICE_ATLAS=instance-$postfix
SERVICE_ATLAS_RENAME=$SERVICE_ATLAS-rnm

BROKER_OSB_SERVICE_NAME="atlas"
