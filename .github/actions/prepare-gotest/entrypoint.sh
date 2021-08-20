#!/bin/bash
# used for adding a mask to some dynamic variable

source ".github/base-dockerfile/helpers/params.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"
make_pcf_metadata "$INPUT_CF_URL" "$INPUT_CF_USER" "$INPUT_CF_PASSWORD"
cf_app_url="api.$(pcf cf-info | grep system_domain | cut -d' ' -f 3)"
cf_app_user="$(pcf cf-info | grep admin_username | cut -d' ' -f 3)"
cf_app_password="$(pcf cf-info | grep admin_password | cut -d' ' -f 3)"

echo "::add-mask::${cf_app_url}"
echo "::add-mask::${cf_app_user}"
echo "::add-mask::${cf_app_password}"

echo "::set-output name=CF_APP_URL::${cf_app_url}"
echo "::set-output name=CF_APP_USER::${cf_app_user}"
echo "::set-output name=CF_APP_PASSWORD::${cf_app_password}"
