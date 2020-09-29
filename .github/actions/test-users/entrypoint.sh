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
password=$(date | md5sum)
user_email="test12323ao@gmail.com"

cf update-service "${SERVICE_ATLAS_RENAME}" -c '{ "op" : "AddUserToProject", "email" : "'"${user_email}"'", "password" : "'"${password}"'"}'
check_service_update "$SERVICE_ATLAS_RENAME"
users=$(get_org_users "${INPUT_ATLAS_ORG_ID}")
status=$(echo "${users}" | awk '/'\"username\"'[: ]*'\"${user_email}\"'/{print "exist"}')
assert_equal "${status}" "exist"
echo "User created"

cf update-service "${SERVICE_ATLAS_RENAME}" -c '{ "op" : "RemoveUserFromProject", "email" : "'"${user_email}"'"}'
check_service_update "$SERVICE_ATLAS_RENAME"
users=$(get_org_users "${INPUT_ATLAS_ORG_ID}")

for elkey in $(echo "$users" | jq '.results | keys | .[]'); do
    user=$(echo "$users" | jq ".results[$elkey]")
    username=$(echo "$user" | jq -r '.username')
    if [[ $username == ${user_email} ]]; then
        for rolekey in $(echo "$user" | jq '.roles | keys | .[]'); do
            el=$(echo "$user" | jq ".roles[$rolekey]")
            orgId=$(echo "$el" | jq -r '.orgId')
            # role=$(echo "$el" | jq -r '.roleName')
            echo "Check if user doesn't have organization in the list"
            assert_not_equal $orgId ${INPUT_ATLAS_ORG_ID}
        done
    fi
done
