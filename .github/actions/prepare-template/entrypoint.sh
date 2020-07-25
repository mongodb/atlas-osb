#!/bin/bash
cd atlas-osb

source ".github/base-dockerfile/helpers/tmp-helper.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"

echo "init"
INSTALL_TIMEOUT=40 #service deploy timeout
branch_name=$(echo $GITHUB_REF | awk -F'/' '{print $3}')
make_pcf_metadata $INPUT_PCF_URL $INPUT_PCF_USER $INPUT_PCF_PASSWORD
make_multikey_config ./apikeys-config.json

echo "Login. Create ORG and SPACE depended on the branch name"
cf_login
cf create-org $ORG_NAME && cf target -o $ORG_NAME
cf create-space $SPACE_NAME && cf target -s $SPACE_NAME

echo "Create service-broker"
cf push $BROKER_APP --no-start
cf set-env $BROKER_APP BROKER_HOST 0.0.0.0
cf set-env $BROKER_APP BROKER_PORT 8080
cf set-env $BROKER_APP BROKER_APIKEYS $(awk NF=NF RS= OFS= < apikeys-config.json)
cf set-env $BROKER_APP ATLAS_BROKER_TEMPLATEDIR ./samples/plans

cf start $BROKER_APP
check_app_started $BROKER_APP
app_url=$(cf app $BROKER_APP | awk '/routes:/{print $2}')
cf create-service-broker $BROKER "admin" "admin" http://$app_url --space-scoped #TODO form

cf marketplace
cf create-service mongodb-atlas-template "override-bind-db-plan"  $SERVICE_ATLAS -c '{"org_id":"'"${INPUT_ATLAS_ORG_ID}"'"}'  #'{"cluster":  {"providerSettings":  {"regionName": "EU_CENTRAL_1"} } }'
check_service_creation $SERVICE_ATLAS

echo "Simple app"
git clone https://github.com/leo-ri/simple-ruby.git
cd simple-ruby
cf push $TEST_SIMPLE_APP --no-start
cf bind-service $TEST_SIMPLE_APP $SERVICE_ATLAS
cf restart $TEST_SIMPLE_APP
check_app_started $TEST_SIMPLE_APP
app_url=$(cf app $TEST_SIMPLE_APP | awk '$1 ~ /routes:/{print $2}')
echo "::set-output name=app_url::$app_url"

# echo "Spring-Music app"
# # cf push APP_NAME --docker-image [REGISTRY_HOST:PORT/]IMAGE[:TAG] [--docker-username USERNAME] [-c COMMAND] [-f MANIFEST_PATH | --no-manifest] [--no-start] [-i NUM_INSTANCES] [-k DISK] [-m MEMORY] [-t HEALTH_TIMEOUT] [-u (process | port | http)] [--no-route | --random-route | --hostname HOST | --no-hostname] [-d DOMAIN] [--route-path ROUTE_PATH] [--var KEY=VALUE]... [--vars-file VARS_FILE_PATH]...
# git clone https://github.com/leo-ri/spring-music.git
# cd spring-music
# chmod +x ./gradlew
# ./gradlew clean assemble
# cf push $TEST_SPRING_APP --no-start
# cf set-env $TEST_SPRING_APP JAVA_OPTS "-Dspring.profiles.active=mongodb"
# cf bind-service $TEST_SPRING_APP $SERVICE_ATLAS
# cf start $TEST_SPRING_APP
# app_url=$(cf app $TEST_SPRING_APP | awk '$1 ~ /routes:/{print $2}')
# echo "::set-output name=app_url::$app_url"
