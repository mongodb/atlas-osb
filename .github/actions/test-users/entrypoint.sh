#!/bin/bash
set -e
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"
source ".github/base-dockerfile/helpers/atlasAPI.sh"
source ".github/base-dockerfile/helpers/asserts.sh"

echo "Login. Create ORG and SPACE depended on the branch name"
cf_login "$ORG_NAME" "$SPACE_NAME"
cf create-org "$ORG_NAME" && cf target -o "$ORG_NAME"
cf create-space "$SPACE_NAME" && cf target -s "$SPACE_NAME"

#check users
user_email="test12323ao@gmail.com"

cf update-service "${SERVICE_ATLAS_RENAME}" -c '{ "op" : "AddUserToProject", "email" : "'"${user_email}"'"}'
check_service_update "$SERVICE_ATLAS_RENAME"
#TODO
users=$(get_org_users "${INPUT_ATLAS_ORG_ID}")
status=$(echo "${users}" | awk '/'\"username\"'[: ]*'\"${user_email}\"'/{print "exist"}')
assert_equal "${status}" "exist"
echo "User created"

cf update-service "${SERVICE_ATLAS_RENAME}" -c '{ "op" : "RemoveUserFromProject", "email" : "'"${user_email}"'"}'
check_service_update "$SERVICE_ATLAS_RENAME"
projectID=$(get_projects | jq '.results[] | select(.name=="'"${SERVICE_ATLAS}"'") | .id')
users=$(get_org_users "${INPUT_ATLAS_ORG_ID}")
project=$(echo "${users}" | jq '.results[] | select(.emailAddress=="'"${user_email}"'") | .roles[] | select(.groupId=='"${projectID}"') ')
assert_equal "${project}" ""
