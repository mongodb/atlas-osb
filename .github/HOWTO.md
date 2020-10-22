# GitHub Actions
We deploy Atlas Broker to Cloud Foundry using [GitHub Actions](https://docs.github.com/en/actions)

# Folder structure description

## action
The folder consists of implemented GitHub actions which are used in "workflows".
Actions available right now:
- clean-failed - triggered by deleting the branch, `purge` failed services and clean test-organization
- cleanup-cf - clean Cloud Foundry space after testing
- e2e-cf - deploy atlas broker with provided templates
- reaper - delete clusters/project from Atlas 

"e2e-cf" action have commented parts in case spring-music are more preferiable as a test application 

## base-dockerfile
The Dockerfile included here is used for actions and also contains helper functions for actions

## disabled-workflows
All disabled/cancelled/saved for future use workflows are placed in here. For example, workflow "Deploy to Amazon ECS" we plan to use it later.

## workflows
Active workflows for operating. 
- `clean-cf.yml` clean Cloud Foundry from previous usage
- `deploy-broker.yml` deploy broker to CF
- `reaper.yml` delete clusters from Atlas
- `create-release-package.yml` create a release
- `k8s-demo-*` demo 
- `test-org-user.yml` copy of `deploy-broker.yml` additionally, it includes a check to create org users by broker

# Using GitHub Actions locally
Tools for successfully running pipeline locally:
- `act` [allow run GitHub action without pushing changes to repo](https://github.com/nektos/act)
- `githubsecrets` allow change/create repository [Github secrets](https://github.com/unfor19/githubsecrets)

## Req. for running act
Put the file `.actrc` to the root project folder with used secrets in GitHub

```
-s ATLAS_BROKER_URL=<url to already deployed broker, used for Deploy to Amazon ECS>
-s ATLAS_PRIVATE_KEY=<private key>
-s ATLAS_PUBLIC_KEY=<public key>
-s ATLAS_ORG_ID=<org_id for the templates>
-s CF_PASSWORD=<password>
-s CF_API=api.something
-s CF_USER=<user>
-s DOCKERHUB_USERNAME=<...>
-s DOCKERHUB_TOKEN=<...>
-s AWS_ACCESS_KEY_ID=<...>
-s AWS_SECRET_ACCESS_KEY=<...>
-s KUBE_CONFIG_DATA=<...one line json kubeconfig...>
```

Now simply call:

```
act delete #call clean-cf workflow
act push #call deploy-broker
act <trigger>
```

!NOTE! deploy-broker workflow deletes installed services at the end. If you need to look at prepared services - uncomment/delete "Cleanup ENV for current branch" step from deploy-broker.yml

Also, `act` can use [event payload](https://developer.github.com/webhooks/event-payloads/#delete) as an argument

```
act delete -e delete.json
```

for running pipeline `test-org-user.yml` we can run a separate job

```
act -j check-users
```

## Demo
demo workflows:
0) `k8s-demo-catalog` install service catalog to k8s cluster to `catalog` namespace, run only if k8s doesn't have a service catalog installed
1) `k8s-demo-broker` deploys broker into k8s cluster, creates service instance, deploys test application. In the end, prints out test application URL
2) `k8s-demo-instance` deploy service instance
3) `k8s-demo-test-app` deploy test application for demonstration
4) `k8s-demo-clean` clean k8s cluster

These jobs accept the `KUBE_CONFIG_DATA` secret, which is a copy of `kubectl config view -o json --raw | jq -c`

```
#sample if `.actrc` file doesn't have `KUBE_CONFIG_DATA` secret or there are many different k8s clusters to use:
act -s KUBE_CONFIG_DATA="$(cat ./kubeconfigoneline.json)" -j k8s-demo-broker
```

Workflows work with default [parameters](https://github.com/mongodb/atlas-osb/blob/master/.github/base-dockerfile/helpers/params.sh), if it is necessary to work with another namespace then a better way is to create an event file with inputs: `service_name` and `namespace`. Samples:

```
echo '{"action":"workflow_dispatch", "inputs": {"service_name":"sky-service","namespace":"atlas-osb"}}' > event.json
act -j eksdemo-broker -e event.json
act -j eksdemo-instance -e event.json
act -j eksdemo-test -e event.json
#NOTE `.actrc` file should include `KUBE_CONFIG_DATA` secret
```
