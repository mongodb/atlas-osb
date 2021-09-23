#!/bin/bash
source ".github/base-dockerfile/helpers/params.sh"
export ORG_NAME=$ORG_NAME
export SPACE_NAME=$SPACE_NAME
export BROKER=$BROKER
export BROKER_APP=$BROKER_APP
export TEST_SIMPLE_APP=$TEST_SIMPLE_APP
export BROKER_APP=$BROKER_APP
export SERVICE_ATLAS=$SERVICE_ATLAS

cd test/cfe2e || exit
ginkgo --failFast --slowSpecThreshold 15 --trace -v -focus "${TEST}"
