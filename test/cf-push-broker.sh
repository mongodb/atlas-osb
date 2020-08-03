KEYS="${1:-./my-test-keys.json}"
BROKER="${2:-atlas-osb}"
echo "KEYS=${KEYS} BROKER=${BROKER}"
cf set-env "${BROKER}" BROKER_LOG_LEVEL DEBUG
cf set-env "${BROKER}" BROKER_HOST 0.0.0.0
cf set-env "${BROKER}" BROKER_APIKEYS "$(cat ${KEYS})"
cf set-env "${BROKER}" ATLAS_BROKER_TEMPLATEDIR samples/plans
cf restage atlas-osb
