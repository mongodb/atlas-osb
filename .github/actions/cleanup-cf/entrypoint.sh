#!/bin/bash

source ".github/base-dockerfile/helpers/tmp-helper.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"

echo "CleanUP: delete service broker, service, unbind app"
INSTALL_TIMEOUT=40 #service deploy timeout
branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
echo $branch_name
org_name="atlas-test-$branch_name"
make_pcf_metadata $INPUT_PCF_URL $INPUT_PCF_USER $INPUT_PCF_PASSWORD

cf_login $org_name $org_name

delete_service_app_if_exists aws-atlas-test-instance-$branch_name simple-app-$branch_name
cf delete-service-broker mongodb-atlas-$branch_name -f
delete_application test-app-$branch_name
delete_application simple-app-$branch_name
delete_application atlas-osb-app-$branch_name
