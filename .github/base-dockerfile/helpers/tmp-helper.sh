#!/usr/local/bin/dumb-init /bin/bash
# shellcheck shell=bash

make_multikey_config() {
	local file=$1
	if [ -f "$file" ]; then
		rm "$file"
	fi
	cat >"$file" <<EOF
{
	"broker": {
		"username": "admin",
		"password": "admin"
	},
	"keys": {
		"testKey": {
			"orgID" : "${INPUT_ATLAS_ORG_ID}",
			"publicKey": "${INPUT_ATLAS_PUBLIC_KEY}",
			"privateKey": "${INPUT_ATLAS_PRIVATE_KEY}"
		}
	}
}

EOF
}

make_creds() {
	mkdir ~/.aws
	cat >>~/.aws/credentials <<EOF
[default]
aws_access_key_id="${AWS_ACCESS_KEY_ID}"
aws_secret_access_key="${AWS_SECRET_ACCESS_KEY}"
EOF
	cat >>~/.aws/config <<EOF
[default]
region=us-east-2
EOF
}