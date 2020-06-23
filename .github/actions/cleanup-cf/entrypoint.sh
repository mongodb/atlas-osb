#!/bin/bash

source ".github/base-dockerfile/helpers/tmp-helper.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"

echo "CleanUP: delete service broker, service, unbind app"
INSTALL_TIMEOUT=30 #service deploy timeout

make_pcf_metadata $INPUT_PCF_URL $INPUT_PCF_USER $INPUT_PCF_PASSWORD

cf_login $ORG_NAME $SPACE_NAME

delete_service_app_if_exists $SERVICE_ATLAS $TEST_SIMPLE_APP
cf delete-service-broker $BROKER -f
delete_application $TEST_SPRING_APP
delete_application $TEST_SIMPLE_APP
delete_application $BROKER_APP
cf delete-service $CREDHUB -f
cf delete-space $SPACE_NAME -f
