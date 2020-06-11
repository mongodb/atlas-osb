branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
#arguments for actions
ORG_NAME="atlas-test-$branch_name"
BROKER=atlas-broker-$branch_name
BROKER_APP=atlas-osb-app-$branch_name
CREDHUB=credhub-$branch_name
TEST_SIMPLE_APP=simple-app-$branch_name
TEST_SPRING_APP=music-$branch_name
SERVICE_ATLAS=aws-atlas-test-instance-$branch_name
