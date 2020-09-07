#!/bin/bash

cd test/cfe2e || exit

ginkgo --failFast --slowSpecThreshold 15 --trace -v
