#!/usr/local/bin/dumb-init /bin/bash
# shellcheck shell=bash disable=SC2155

cf_login() {
  local org=$1
  local space=$2

  if [[ -z $org ]]; then
    org="system"
  fi
  if [[ -z $space ]]; then
    space="system"
  fi

  cf login -a "$INPUT_CF_API" -u "$INPUT_CF_USER" -p "$INPUT_CF_PASSWORD" --skip-ssl-validation -o $org
  if [[ $org == "system" && $space == "system" ]]; then
    cf create-space ${space} -o ${org}
  fi

  cf target -o ${org} -s ${space}
}

#cf.helper. wait for particular service status
wait_service_status_change() {
  local instance_name=$1
  local status=$2
  local time=0
  echo "checking " "$instance_name" "$status"
  local verify_status
  verify_status=$(cf services | awk '/'"$instance_name"'[ ].*'"$status"'/{print "'"$status"'"}')
  while [[ $verify_status == "$status" ]] && [[ $time -lt $INSTALL_TIMEOUT ]]; do
    echo "...${verify_status}"
    sleep 3m
    ((time = time + 3))
    verify_status=$(cf services | awk '/'"$instance_name"'[ ].*'"$status"'/{print "'"$status"'"}')
  done
}

delete_service_app_if_exists() {
  local instance_name=$1
  local app_name=$2
  delete_bind "$instance_name" "$app_name"
  delete_application "$app_name"
  delete_service "$instance_name" "$app_name"
}

delete_bind() {
  local instance_name=$1
  local app_name=$2
  echo "check if $app_name binding exist"
  local binding
  binding=$(cf services | grep "$instance_name" | awk '/'"$app_name"'/{print "exist"}')
  if [[ $binding == "exist" ]]; then
    cf unbind-service "$app_name" "$instance_name"
    check_app_unbinding "$instance_name" "$app_name"
  fi
}

delete_service() {
  local instance_name=$1
  local app_name=$2
  local service
  service=$(cf services | awk '/'"$instance_name"'[ $]/{print "exist"}')
  if [[ $service == "exist" ]]; then
    cf delete-service "$instance_name" -f
    wait_service_status_change "$instance_name" "delete in progress"
    service_status=$(cf services | awk '/'"$instance_name"'[ $].*failed/{print "failed"}')
    if [[ $service_status == "failed" ]]; then
      cf purge-service-instance "$instance_name" -f
    fi
  fi
}

delete_application() {
  local app_name=$1
  echo "check if $app_name exists"
  local app
  app=$(cf apps | awk '/'"$app_name"'[ $]/{print "exist"}')
  if [[ $app == "exist" ]]; then
    cf delete "$app_name" -f
  fi
}

create_atlas_service_broker_from_repo() {
  local broker_name=$1
  local app_name=$2 #atlas-osb-app
  cf push "$app_name" --no-start
  cf set-env "$app_name" BROKER_HOST 0.0.0.0
  cf set-env "$app_name" BROKER_PORT 8080
  cf start "$app_name"
  check_app_started "$app_name"
  app_url=$(cf app "$app_name" | awk '/routes:/{print $2}')
  cf create-service-broker "$broker_name" "$INPUT_ATLAS_PUBLIC_KEY"@"$INPUT_ATLAS_PROJECT_ID" "$INPUT_ATLAS_PRIVATE_KEY" http://"$app_url" --space-scoped
}

create_atlas_service_broker_from_ECS() { #TODO
  local broker_name=$1
  cf create-service-broker "$broker_name" "$INPUT_ATLAS_PUBLIC_KEY"@"$INPUT_ATLAS_PROJECT_ID" "$INPUT_ATLAS_PRIVATE_KEY" "$INPUT_ATLAS_BROKER_URL" --space-scoped
}

#credhub multikeys #TODO clean up after final solution
create_service() {
  local instance_name=$1 #aws-atlas-test-instance-$INPUT_BRANCH_NAME
  local plan=$2
  #local config=$2
  cf create-service mongodb-atlas-aws "$plan" "$instance_name" -c '{"cluster":  {"providerSettings":  {"regionName": "EU_CENTRAL_1"} } }'
  wait_service_status_change "$instance_name" "create in progress"
  service_status=$(cf services | awk '/'"$instance_name"'[ ].*succeeded/{print "succeeded"}')
  if [[ $service_status != "succeeded" ]]; then
    echo "FAILED! wrong status: $(cf service "$instance_name")"
    cf logout
    exit 1
  fi
}

check_service_creation() {
  local instance_name=$1
  wait_service_status_change "$instance_name" "create in progress"
  service_status=$(cf services | awk '/'"$instance_name"'[ ].*succeeded/{print "succeeded"}')
  if [[ $service_status != "succeeded" ]]; then
    echo "FAILED! wrong status: $(cf service "$instance_name")"
    cf logout
    exit 1
  fi
}

check_service_update() {
  local instance_name=$1
  wait_service_status_change "$instance_name" "update in progress"
  service_status=$(cf services | awk '/'"$instance_name"'[ ].*succeeded/{print "succeeded"}')
  if [[ $service_status != "succeeded" ]]; then
    echo "FAILED! wrong status: $(cf service "$instance_name")"
    cf logout
    exit 1
  fi
}

check_app_unbinding() {
  local instance_name=$1
  local app_name=$2
  local app_binding
  app_binding=$(cf services | grep "$instance_name " | awk '!/'"$app_name"'/{print "not bounded"}')
  local try=10
  until [[ $app_binding == "not bounded" ]]; do
    app_binding=$(cf services | grep "$instance_name " | awk '!/'"$app_name"'/{print "not bounded"}')
    if [[ $try -lt 0 ]]; then
      echo "ERROR: unbinding is getting too long"
      exit 1
    fi
    ((try--))
    echo "checking unbinding ($try)"
  done
}

check_app_started() {
  local app_name=$1
  local app
  app=$(cf app "$app_name" | tail -1 | awk '{print $2}')
  local try=10
  until [[ $app == "running" ]]; do
    app=$(cf app "$app_name" | tail -1 | awk '{print $2}')
    if [[ $try -lt 0 ]]; then
      echo "ERROR: startup is getting too long"
      cf logs "$app_name" --recent | tail -25
      exit 1
    fi
    ((try--))
    echo "checking application status: $app ($try)"
  done
  echo "Application started"
}
