# Getting started with the atlas-osb on Cloud Foundry

The atlas-osb installs into Cloud Foundry with a usual `cf push` command. It does not require a tile to be installed into PCF Ops Manager. These instuctions show a basic installation, configuration, and use of the broker. These have been tested on PCF 2.9. 

### PLEASE NOTE 
These instuctions only apply to **this** the new template-based atlas-osb and **_not_** the legacy [MongoDB Atlas Service Broker](https://github.com/mongodb/mongodb-atlas-service-broker).

This software is in active development. 

## Prereq's

Typical cli/bash commands are used for this procedure along with the following tools:

* `cf` Cloud Foundry cli
* `hammer` cli (recommended)
* text editor
* MongoDB Atlas account - you need an Org apikey

Once you have your workstation ready, head over to http://cloud.mongodb.com and create an organization apikey. For more step by step details on this step, step over here: https://docs.atlas.mongodb.com/configure-api-access/#manage-programmatic-access-to-an-organization.

## Installation & Configuration

1. Pull down the latest release of the atlas-osb (recommended).

```bash
curl -OL https://github.com/jasonmimick/atlas-osb/releases/download/v0.1-alpha/atlas-osb-v0.1-alpha.tar.gz
tar xvf atlas-osb-latest.tar.gz
cd atlas-osb
```

or, you can clone this repo.

2. Setup apikeys. 
This flow uses a simple User Provided Service to store the apikey for the broker. The broker also supports CredHub integration [TODO: add section].

Copy this template and update with your own apikey information.

```json
{
   "broker": {
      "username": "admin",
      "password": "admin",
      "db": "mongodb+srv://jason:jason@statestorage-mytsp.mongodb.net/admin?retryWrites=true&w=majority"
   },
   "orgs": {
      "<ORG-ID>": {
        "publicKey": "<PUBLIC-KEY>",
        "privateKey": "<PRIVATE-KEY>",
        "id": "my-test-key",
        "desc": "My first key for the atlas-osb.",
        "roles": [
            { "orgId" : "<ORG-ID>" }
        ]
      }
   }
}
```

Save this in a file called `keys`.

__NOTE__ The `broker.db` field in the json above currently holds a connection string to another Atlas cluster we are using to store data on the mappings between serivce instance id's and the Atlas project, cluster, etc ids. The use of another external db will __NOT__ be a requirement of the GA version of atlas-osb. This is only a temporary solution, to be addressed in Sprint 3.

3. Create a user provided service:

```bash
# create simple user-provided-service with keys file
cf cups atlas-osb-keys -p keys
```


4. Installing the service broker app

The initial setup includes deploying the cf app and configuring some environment variables.
By default, the broker will only listen on localhost (127.0.0.1) and port 4000.
Use the `set-env` command to update this for Cloud Foundry defaults (port 8080).

```bash
# push the cf app for the broker
cf push atlas-osb --no-start

# connect the keys to the broker & configure settings
cf bind-service atlas-osb atlas-osb-keys

# use the provided sample plan templates
cf set-env atlas-osb ATLAS_BROKER_TEMPLATEDIR ./samples/plans

cf set-env atlas-osb BROKER_HOST 0.0.0.0
cf set-env atlas-osb BROKER_PORT 8080

# start it up
cf start atlas-osb
```bash

Check the logs, you should see output included the loaded plan templates.

```bash
cf logs atlas-osb --recent
```

4. Register the app as a real service broker

Grab the `routes` value from `cf app atlas-osb` and use this URL in the following command.

```bash
# create the actual broker
cf create-service-broker atlas-osb admin admin <YOUR-DEPLOYMENT-URL>
# enable access
cf enable-service-access mongodb-atlas-template
```

Check out the results.

```bash
cf marketplace
cf apps
```

5. Create an Atlas cluster

For this flow, we'll create an instance of the "basic-plan" called "hello-atlas-osb".
See [/samples/plans/sample_basic.yml.tpl](/samples/plans/sample_basic.yml.tpl).

Use your own <YOUR-ORG-ID> in the following command.

```bash
cf create-service mongodb-atlas-template basic-plan hello-atlas-osb -c '{"org_id":"<YOUR-ORG-ID>"}'
```

Head over to http://cloud.mongodb.com to follow progress as your Atlas resources are provisioning.

6. (Optional) You can test connecting to your new Atlas cluster with a very simple app in package.

Push, bind, and test sample app.

```bash
cd test/hello-atlas-cf
cf push hello-atlas-cf --no-start
cf bind-service hello-atlas-cf hello-atlas-osb
cf start hello-atlas-cf
cf app hello-atlas-cf
```

Grab the route and load it up in your browser. You should see connection information to your new Atlas cluster.

## Advanced

You can inspect more scenarios over in our Github [actions](/.github/actions) and [workflows](./github/workflows). 

### [How to bind and connect to a specific database](#how-to-bind-db)

You need to override the role when creating the binding like this:

```bash
 cf bind-service hello-atlas-cf wed-demo-1 -c '{"User" : {"roles" : [ { "roleName" : "readWrite", "databaseName" : "default"} ] } }'
```

You can check this works with the hello-atlas-cf app with something like this:

```bash
cf app hello-atlas-cf | grep routes | cut -d: -f2 | xargs -I {} google-chrome "{}"
```

### Pausing a cluster

```bash
 cf update-service <SERVICE-INSTANCE-NAME> -c '{ "paused":true }'
```

Note - do not put quotes around the true/value.

### Updateing a cluster

First - note not all possible updates are supported at this time.

In general, to update a plan instance:

```
 cf create-service -b dyno-ss mongodb-atlas-template basic-plan dyno-ss-4oh -c '{"org_id":"5ea0477597999053a5f9cbec", "cluster" :  { "mongoDBMajorVersion": "4.0" } }'

Creating service instance dyno-ss-4oh in org atlas-broker-demo / space dynoss as admin...
OK
```

The syntax for the -c json you send in create-service or update-service is processed in two ways:

1. The document passed is parsed and matched into the template dot-variables, then the template is executed.
2. The passed document treated as a partial plan-instance and then merged into the results from step 1.

This allows service settings to be updated.

TODO: Add flag in Plan's to enable this feature, default should be False.

### mongocli

`mongocli` is a tool from MongoDB, https://github.com/mongodb/mongocli.
It can be useful, for example you can check the state of a cluster like this,

Grab the `projectId` from the dashboard url:

```bash
cf service wed-demo-1
Showing info of service wed-demo-1 in org atlas-broker-demo / space test1 as admin...

name:             wed-demo-1
service:          mongodb-atlas-template
tags:             
plan:             basic-plan
description:      MonogoDB Atlas Plan Template Deployments
documentation:    https://support.mongodb.com/welcome
dashboard:        https://cloud.mongodb.com/v2/5f0f0e0859cd8a08718294bf#clusters/detail/wed-demo-1
service broker:   atlas-osb
...
```

then, once you setup `mongocli` [TODO - integration with atlas-osb keys]

```bash
mongocli atlas clusters list --projectId 5f0f0e0859cd8a08718294bf
```

Use tools like `jq` to do fun stuff,

```bash
 mongocli atlas clusters list --projectId 5f0f0e0859cd8a08718294bf | jq '.[] | { "n" : .name, "s" : .paused}'
{
  "n": "wed-demo-1",
  "s": false
}
```
