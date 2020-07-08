branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
commit_id=$(git rev-parse --short HEAD)
postfix=$branch_name-$commit_id

#arguments for actions
ORG_NAME="atlas-test-$branch_name"
SPACE_NAME=$commit_id
BROKER=atlas-broker-$postfix
BROKER_APP=atlas-osb-app-$postfix
CREDHUB=credhub-$postfix
TEST_SIMPLE_APP=simple-app-$postfix
TEST_SPRING_APP=music-$postfix
SERVICE_ATLAS=aws-atlas-test-instance-$postfix
