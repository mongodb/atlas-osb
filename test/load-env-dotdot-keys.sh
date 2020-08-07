#!/usr/bin/env bash

echo "echo \"Loading ATLAS_* env from first org in ../keys\""
echo "export ATLAS_PUBLIC_KEY=\$(cat ../keys | jq '.orgs | .[] | .publicKey')"
echo "export ATLAS_PRIVATE_KEY=\$(cat ../keys | jq '.orgs | .[] | .privateKey')"
echo "export ATLAS_GROUP_ID=\$(cat ../keys | jq '.orgs | .[] | .roles[0].orgId')"

