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

make_sample_credhub_config() {
  local file=$1
  if [ -f $file ]; then
      rm $file
  fi
  cat >$file <<EOF
{
	"broker": {
		"username": "admin",
		"password": "admin"
	},
	"projects": {
		"${INPUT_ATLAS_PROJECT_ID}": {
			"public_key": "${INPUT_ATLAS_PUBLIC_KEY}",
			"api_key": "${INPUT_ATLAS_PRIVATE_KEY}",
			"display_name": "valley"
		},
		"${INPUT_ATLAS_PROJECT_ID_BAY}": {
			"public_key": "${INPUT_ATLAS_PUBLIC_KEY}",
			"api_key": "${INPUT_ATLAS_PRIVATE_KEY}",
			"display_name": "bay"
		}
	}
}
EOF
}
