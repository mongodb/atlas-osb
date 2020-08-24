#!/bin/bash
# shellcheck shell=bash disable=SC2155 disable=SC1091

# Will delete PROJECT AND CLUSTER IN ORGANIZATION except M0 and project ValleyOfTesing
# doesn't wait cluster termination for deleting project

set -e
source ".github/base-dockerfile/helpers/atlasAPI.sh"

projects=$(get_projects)

for elkey in $(echo "$projects" | jq '.results | keys | .[]'); do
    element=$(echo "$projects" | jq ".results[$elkey]")
    count=$(echo "$element" | jq -r '.clusterCount')
    id=$(echo "$element" | jq -r '.id')
    name=$(echo "$element" | jq -r '.name')

    if [[ $count != 0 ]]; then
        clusters=$(get_clusters "$id")

        #check cluster size, if it is not M0 - delete.
        for ckey in $(echo "$clusters" | jq '.results | keys | .[]'); do
            cluster=$(echo "$clusters" | jq -r ".results[$ckey]")
            csize=$(echo "$cluster" | jq -r '.providerSettings.instanceSizeName')
            cname=$(echo "$cluster" | jq -r '.name')
            if [[ $csize != "M0" ]]; then
                echo "delete cluster: $id $cname $csize"
                delete_cluster "$id" "$cname"
                #not going to wait for deleting projects
            else 
                echo "$cname $csize is M0"
            fi
        done
    else 
        if [[ $name != "ValleyOfTesting" ]]; then
            echo "deleting project: $id"
            delete_project "$id"
        else
            echo "won't delete ValleyOfTesting"
        fi
    fi
done 


