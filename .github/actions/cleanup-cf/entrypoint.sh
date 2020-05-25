#!/bin/bash

source "/home/tmp-helper.sh"
source "/home/cf-helper.sh"

echo "CleanUP: delete service broker, service, unbind app"
INSTALL_TIMEOUT=40 #service deploy timeout
branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
#branch_name=$(echo $GITHUB_REF | awk -F'\' '{print $4}') #TODO windows
echo $branch_name
org_name="atlas-test-org"
make_pcf_metadata $INPUT_PCF_URL $INPUT_PCF_USER $INPUT_PCF_PASSWORD

cf_login
space_name="test-space"-$branch_name
cf target -o $org_name -s $space_name

cf unbind-service test-app-$branch_name aws-atlas-test-instance-$branch_name
cf delete-service aws-atlas-test-instance-$branch_name -f
wait_service_status_change aws-atlas-test-instance-$branch_name "delete in progress"
cf delete-service-broker mongodb-atlas-$branch_name -f
delete_application test-app-$branch_name
delete_application atlas-osb-app-$branch_name
