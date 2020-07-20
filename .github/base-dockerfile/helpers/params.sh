branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
branch_name=${branch_name:0:30} #service name max length is 50 symbols minus prefixes
commit_id=$(git rev-parse --short HEAD)
postfix=$branch_name-$commit_id

#arguments for actions
ORG_NAME="atlas-test-$branch_name"
SPACE_NAME=$commit_id
BROKER=atlas-broker-$postfix
BROKER_APP=aosb-app-$postfix
CREDHUB=credhub-$postfix
TEST_SIMPLE_APP=simple-app-$postfix
TEST_SPRING_APP=music-$postfix
SERVICE_ATLAS=instance-$postfix
