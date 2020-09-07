#!/bin/bash
# shellcheck disable=SC1091,SC2034

cd test/cfe2e || exit
#DO NOT run "ginkgo -v " 
ginkgo --failFast
