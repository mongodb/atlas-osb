# GitHub Actions
GitHub Actions help to automate and customize workflows. We deploy Atlas Broker to Cloud Foundry using [GitHub Actions](https://docs.github.com/en/actions). Also, they are used for other tasks like release Atlas-OSB, cleaning our Atlas organization, testing, and demo-runs.

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

# Using GitHub Actions locally
Tools for successfully running pipeline locally:
- `act` allows running GitHub actions without pushing changes to a repository, more information [here](https://github.com/nektos/act)
- `githubsecrets` helps us to change/create [Github secrets](https://github.com/unfor19/githubsecrets) from CLI

## Requirements for running actions with the act tool
It is necessary to have a file named `.actrc` in the root of the project folder with the same secrets names mentioned in workflows.
Below there is an actual sample of `.actrc` with all required secrets:

```
-s ATLAS_BROKER_URL=<url to already deployed broker, used for Deploy to Amazon ECS>
-s ATLAS_PRIVATE_KEY=<private key>
-s ATLAS_PUBLIC_KEY=<public key>
-s ATLAS_ORG_ID=<org_id for the templates>
-s CF_PASSWORD=<password>
-s CF_URL=<https://pcf.host>
-s CF_USER=<user>
-s REGISTRY=quay.io
-s REGISTRY_USERNAME=<...>
-s REGISTRY_PASSWORD=<...>
-s REGISTRY_REPO=test/test
-s KUBE_CONFIG_DATA=<...one line json kubeconfig...>
-s SENTRY_DSN=http://setry.host
```

Now simply call:

```bash
#act <trigger>
act delete #call clean-cf workflow
act push #call deploy-broker
```

This sample runs a specific job

```bash
act -j <job name> #call job
```

Additionally, we can run workflows/jobs with different runs-on images:

```bash
act -j k8s-demo-broker -P ubuntu-latest=leori/atlas-ci:v2
```

!NOTE! `deploy-broker` workflow deletes installed services at the end. If you need to look at deployed services - just run a separate job from `deploy-broker` workflow:

```bash
act -j basic-plan-cloudfoundry
```

Also, `act` can use [event payload](https://developer.github.com/webhooks/event-payloads/#delete) as an argument

```bash
act delete -e delete.json
```

delete.json sample:
```json
{
  "ref": "some-ref",
  "ref_type": "branch"
}
```

Some workflows have `workflow_dispatch` trigger - manual launch with inputs. It is possible to run it with `act` too. Just create a payload with all inputs:
event.json for k8s-demo-*

```json
{
	"action":"workflow_dispatch",
	"inputs": {
		"service_name":"second-service",
		"namespace":"atlas-k8s-sample-2ba27a6"
	}
}
```

After, run command:

```bash
act -e event.json -j k8s-demo-instance
```

for running pipeline `test-org-user.yml` we can run a separate job

```bash
act -j check-users
```

## About .github/ folder structure description

### action/
The folder consists of implemented GitHub actions which are used in "workflows/".
Actions available right now:
- clean-failed - triggered by deleting the branch, `purge` failed services and clean test-organization
- cleanup-cf - clean Cloud Foundry space after testing
- e2e-cf - deploy atlas broker with provided templates
- reaper - delete all clusters/projects from Atlas, except M0
- gotest - run CF e2e tests

### base-dockerfile/
The Dockerfile included here is used for actions and, also contains helper functions

### disabled-workflows/
All disabled/canceled/saved workflows are placed in here for future use. For example, workflow "Deploy to Amazon ECS," we plan to use it later.

### workflows/
Active workflows for operating.
- `clean-cf.yml` clean Cloud Foundry from previous usage
- `deploy-broker.yml` deploy broker to CF
- `reaper.yml` delete clusters from Atlas
- `create-release-package.yml` create a release
- `k8s-demo-*` demo
- `test-org-user.yml` copy of `deploy-broker.yml` additionally, it includes a check to create org users by broker


## Demo
demo workflows:

0) `k8s-demo-catalog` install service catalog to k8s cluster to `catalog` namespace, run only if k8s doesn't have a service catalog installed
1) `k8s-demo-broker` deploys broker into k8s cluster, creates service instance, deploys test application. At the end, prints out test application URL
2) `k8s-demo-instance` deploy service instance
3) `k8s-demo-test-app` deploy test application for demonstration
4) `k8s-demo-clean` clean k8s cluster

These jobs accept the `KUBE_CONFIG_DATA` secret, for example:

```bash
KUBE_CONFIG_DATA=$(kubectl config view -o json --raw | jq -c '.')
```

If `.actrc` file doesn't have `KUBE_CONFIG_DATA` secret or there are many different k8s clusters to use:

```bash
act -s KUBE_CONFIG_DATA="$(cat ./kubeconfigoneline.json)" -j k8s-demo-broker
```

Workflows work with default [parameters](https://github.com/mongodb/atlas-osb/blob/master/.github/base-dockerfile/helpers/params.sh), if it is necessary to work with another namespace then a better way is to create an event file with inputs: `service_name` and `namespace`. Samples:

```bash
echo '{"action":"workflow_dispatch", "inputs": {"service_name":"sky-service","namespace":"atlas-osb"}}' > event.json
act -j eksdemo-broker -e event.json
act -j eksdemo-instance -e event.json
act -j eksdemo-test -e event.json
```

## Run test sample

```bash
act -j gotest
```
