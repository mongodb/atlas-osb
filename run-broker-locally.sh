#!/usr/bin/env bash

export BROKER_LOG_LEVEL=DEBUG
export BROKER_HOST=0.0.0.0
export BROKER_PORT=4000
export BROKER_APIKEYS=$(cat ./keys)
export ATLAS_BROKER_TEMPLATEDIR=$(pwd)/samples/plans
env
ls -l plans
env | grep BROKER

./mongodb-atlas-service-broker

