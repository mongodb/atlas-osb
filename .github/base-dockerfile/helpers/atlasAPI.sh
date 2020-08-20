#ATLAS API documentation https://docs.atlas.mongodb.com/reference/api/projects/
#shellcheck shell=bash

BASE_URL="https://cloud.mongodb.com/api/atlas/v1.0"

get_projects() {
    curl -s -u "${INPUT_PUBLIC_KEY}:${INPUT_PRIVATE_KEY}" --digest "${BASE_URL}/groups"
}

get_clusters() {
    projectID=$1
    curl -s -u "${INPUT_PUBLIC_KEY}:${INPUT_PRIVATE_KEY}" --digest "${BASE_URL}/groups/${projectID}/clusters"
}

delete_cluster() {
    projectID=$1
    name=$2
    echo "${BASE_URL}/groups/${projectID}/clusters/${name}"
    curl -s -u "${INPUT_PUBLIC_KEY}:${INPUT_PRIVATE_KEY}" --digest -X DELETE "${BASE_URL}/groups/${projectID}/clusters/${name}"
}
delete_project() {
    projectID=$1
    curl -s -X DELETE --digest -u "${INPUT_PUBLIC_KEY}:${INPUT_PRIVATE_KEY}" "${BASE_URL}/groups/${projectID}"
}

get_cluster_info() {
    projectID=$1
    name=$2
    curl -s -u "${INPUT_PUBLIC_KEY}:${INPUT_PRIVATE_KEY}" --digest \
        --header "Content-Type: application/json" \
        --include \
        --request GET "${BASE_URL}/groups/${projectID}/clusters/${name}?pretty=true"
}
