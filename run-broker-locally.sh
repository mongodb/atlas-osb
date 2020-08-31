#!/usr/bin/env bash
#shellcheck shell=bash disable=SC2155

export BROKER_LOG_LEVEL=DEBUG
export BROKER_HOST=0.0.0.0
export BROKER_PORT=4000
export BROKER_APIKEYS=./my-keys.json
export ATLAS_BROKER_TEMPLATEDIR=$(pwd)/samples/plans

./atlas-osb
