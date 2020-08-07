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

make_multikey_config() {
  local file=$1
  if [ -f $file ]; then
      rm $file
  fi
  cat >$file <<EOF
{
	"broker": {
		"username": "admin",
		"password": "admin",
		"db": "${INPUT_BROKER_DB_CONNECTION_STRING}"
	},
	"projects": {
		"${INPUT_ATLAS_PROJECT_ID}": {
			"privateKey": "${INPUT_ATLAS_PRIVATE_KEY}",
			"desc": "valley",
			"publicKey": "${INPUT_ATLAS_PUBLIC_KEY}"
		},
		"${INPUT_ATLAS_PROJECT_ID_BAY}": {
			"publicKey": "${INPUT_ATLAS_PUBLIC_KEY}",
			"privateKey": "${INPUT_ATLAS_PRIVATE_KEY}",
			"desc": "bay"
		}
	},
	"orgs": {
		"${INPUT_ATLAS_ORG_ID}": {
			"publicKey": "${INPUT_ATLAS_PUBLIC_KEY}",
			"privateKey": "${INPUT_ATLAS_PRIVATE_KEY}",
			"desc": "org-key",
            "roles": [
                { "orgid" : "${INPUT_ATLAS_ORG_ID}" }
            ]
		}
	}
}

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
			"desc": "valley"
		},
		"${INPUT_ATLAS_PROJECT_ID_BAY}": {
			"public_key": "${INPUT_ATLAS_PUBLIC_KEY}",
			"api_key": "${INPUT_ATLAS_PRIVATE_KEY}",
			"desc": "bay"
		}
	}
}
EOF
}
