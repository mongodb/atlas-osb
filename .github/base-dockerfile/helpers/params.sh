#shellcheck shell=bash disable=SC2034

branch_name=$(echo "$GITHUB_REF" | awk -F'/' '{print $3}')
branch_name=${branch_name:0:30} #service name max length is 50 symbols minus prefixes
branch_name=$(echo "${branch_name}" | tr "." "-")
commit_id=$(git rev-parse --short HEAD)
postfix=$branch_name-$commit_id

#arguments for actions
ORG_NAME="atlas-test-$branch_name"
SPACE_NAME=$commit_id
BROKER=atlas-osb-$postfix
BROKER_APP=atlas-osb-app-$postfix
CREDHUB=credhub-$postfix
TEST_SIMPLE_APP=simple-app-$postfix
TEST_SPRING_APP=music-$postfix
SERVICE_ATLAS=instance-$postfix

BROKER_OSB_SERVICE_NAME="atlas"
