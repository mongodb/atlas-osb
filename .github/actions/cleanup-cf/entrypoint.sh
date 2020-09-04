#!/bin/bash
# shellcheck disable=SC1091,SC2034
source ".github/base-dockerfile/helpers/tmp-helper.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"

INSTALL_TIMEOUT=30

echo "CleanUP: delete service broker, service, unbind app"
cf_login "$ORG_NAME" "$SPACE_NAME"

delete_bind "$SERVICE_ATLAS_RENAME" "$TEST_SIMPLE_APP"
delete_bind "$SERVICE_ATLAS_RENAME" "$TEST_SPRING_APP"
delete_service_app_if_exists "$SERVICE_ATLAS_RENAME" "$TEST_SIMPLE_APP"
cf delete-service-broker "$BROKER" -f
delete_application "$TEST_SPRING_APP"
delete_application "$TEST_SIMPLE_APP"
delete_application "$BROKER_APP"
cf delete-service "$CREDHUB" -f
cf delete-space "$SPACE_NAME" -f
