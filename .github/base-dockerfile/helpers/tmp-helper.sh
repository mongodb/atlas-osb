#!/usr/local/bin/dumb-init /bin/bash

make_pcf_metadata() {
    local PCF_URL=$1
    local PCF_USERNAME=$2
    local PCF_PASSWORD=$3
    file="metadata"
    if [ -f $file ]; then
        rm $file
    fi
    cat >$file <<EOF
---
opsmgr:
  url: "${PCF_URL}"
  username: "${PCF_USERNAME}"
  password: "${PCF_PASSWORD}"
EOF
}
