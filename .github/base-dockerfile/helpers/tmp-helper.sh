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
