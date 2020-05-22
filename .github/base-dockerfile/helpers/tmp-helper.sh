#!/usr/local/bin/dumb-init /bin/bash

#save config.pie from pipeline
make_env_config() {
    local config=$1
    file=$1
    if [ -f $file ] ; then
        rm $file
    fi
    echo "$CONFIG" >> $file
}

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


