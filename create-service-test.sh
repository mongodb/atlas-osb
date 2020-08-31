#!/usr/bin/env bash
set -ex
N=$(mktemp | cut -d. -f2)
#./broker-tester -op create-service --plan basic-plan --values "{ \"instance_name\" : \"${N}\" }" --name "${N}"

request=$(mktemp)
cat << EOF > "${request}"
{
    "service_id": "aosb-cluster-service-template",
    "plan_id": "aosb-cluster-plan-template-basic-plan",
    "instance_id" : "${N}",
    "organization_guid": "fake",
    "space_guid": "fake",
    "accepts_incomplete": "true",
    "parameters" : {
        "instance_name" : "${N}"
    }
}
EOF


echo "Sending request:"
cat ${request}

curl -X PUT -u admin:admin \
 --header "Content-Type: application/json" \
 --data @${request} \
 "localhost:4000/v2/service_instances/${N}?accepts_incomplete=true"

