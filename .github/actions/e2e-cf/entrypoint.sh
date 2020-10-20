#!/bin/bash
# shellcheck disable=SC1091,SC2034
source ".github/base-dockerfile/helpers/tmp-helper.sh"
source ".github/base-dockerfile/helpers/cf-helper.sh"
source ".github/base-dockerfile/helpers/params.sh"

INSTALL_TIMEOUT=40 #service deploy timeout

echo "init"
make_multikey_config samples/apikeys-config.json

echo "Login. Create ORG and SPACE depended on the branch name"
cf_login "$ORG_NAME" "$SPACE_NAME"
cf create-org "$ORG_NAME" && cf target -o "$ORG_NAME"
cf create-space "$SPACE_NAME" && cf target -s "$SPACE_NAME"

echo "Create service-broker"
cf push "$BROKER_APP"
check_app_started "$BROKER_APP"
app_url=$(cf app "$BROKER_APP" | awk '/routes:/{print $2}')

cf create-service-broker "$BROKER" "admin" "admin" http://"$app_url" --space-scoped #TODO form

cf marketplace
BROKER_OSB_SERVICE_NAME=$(echo "${BROKER_OSB_SERVICE_NAME}" | tr "." "-")
cf create-service "$BROKER_OSB_SERVICE_NAME" "basic-overrides-plan" "$SERVICE_ATLAS" -c '{"org_id":"'"${INPUT_ATLAS_ORG_ID}"'"}' #'{"cluster":  {"providerSettings":  {"regionName": "EU_CENTRAL_1"} } }'
check_service_creation "$SERVICE_ATLAS"

echo "Simple app"
git clone https://github.com/leo-ri/simple-ruby.git
cd simple-ruby || exit 1
cf push "$TEST_SIMPLE_APP" --no-start
cf bind-service "$TEST_SIMPLE_APP" "$SERVICE_ATLAS"
cf restart "$TEST_SIMPLE_APP"
check_app_started "$TEST_SIMPLE_APP"
app_url=$(cf app "$TEST_SIMPLE_APP" | awk '$1 ~ /routes:/{print $2}')
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
# curl -H "Content-Type: application/json" -X PUT -d '{"_class":"org.cloudfoundry.samples.music.domain.Album", "artist": "Tenno", "title": "Journey", "releaseYear": "2019", "genre": "chillhop" }' $app_url/albums
# result=$(curl -X GET $app_url/albums -s | awk '/Tenno/{print "true"}')
# echo $result
# if [[ -z $result ]]; then
#   echo "FAILED. Curl check: Text is not found"
#   exit 1
# fi

echo "Checking test app"
app_url="${app_url}/service/mongo/test3"
data='{"data":"sometest130"}'
status=$(curl -s -X PUT -d "${data}" "${app_url}")
if [[ $status != "success" ]]; then
    echo "Error: can't perform PUT request"
    exit 1
fi
result=$(curl -s -X GET "${app_url}")
if [ "${result}" == "${data}" ]; then
    echo "Application is working"
else
    echo "GET ${app_url} has result: ${result}"
    echo "FAILED. Application doesn't work. Can't get data from DB"
    exit 1
fi

cf rename-service "$SERVICE_ATLAS" "$SERVICE_ATLAS_RENAME"

echo "Updating service"
cf update-service "${SERVICE_ATLAS_RENAME}" -c '{"instance_size":"M20"}'
check_service_update "$SERVICE_ATLAS_RENAME"

echo "Check that saved data still exists"
result=$(curl -s -X GET "${app_url}")
if [ "${result}" == "${data}" ]; then
    echo "Data retrieved after update"
    curl -X DELETE "${app_url}"
else
    echo "GET ${app_url} has result: ${result}"
    echo "FAILED. Can not get data after update"
    exit 1
fi
