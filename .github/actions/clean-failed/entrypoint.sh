#!/bin/bash
# shellcheck disable=SC1091,SC2119
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"

#final cleaning
if [[ "$TYPE" == "branch" ]]; then
    echo "$TYPE $BRANCH has been deleted"

    cf_login
    cf target -o "$ORG_PREFIX$BRANCH"

    empty=$(cf spaces | awk '/No spaces found/{print "true"}')
    if ! "$empty"; then
    echo "not empty"
        for space in $(cf spaces | tail -n+4)
        do
            cf delete-space "$space" -f
        done
    fi

    #the rest spaces have problems and should be purged
    empty=$(cf spaces | awk '/No spaces found/{print "true"}')
    if ! "$empty"; then
        for space in $(cf spaces | tail -n+4)
        do
            cf target -s "$space"
            list=$(cf services | tail -n+4 | awk '{print $1}')
            for service in $list
            do
                cf purge-service-instance "$service" -f
            done
            cf delete-space "$space" -f
        done
    fi

    cf delete-org "$ORG_PREFIX$BRANCH" -f
fi
