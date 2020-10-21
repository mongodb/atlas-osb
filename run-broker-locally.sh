#!/usr/bin/env bash
#shellcheck shell=bash disable=SC2155

export BROKER_LOG_LEVEL=DEBUG
export BROKER_HOST=0.0.0.0
export BROKER_PORT=4000
export "$(grep -v "#.*" ~/local.env | xargs)" && envsubst < ./samples/apikeys-config.json.sample > ./keys
export BROKER_APIKEYS=./keys
export ATLAS_BROKER_TEMPLATEDIR=$(pwd)/samples/plans
tree "${ATLAS_BROKER_TEMPLATEDIR}"
env | grep BROKER
cat keys

./atlas-osb
