# Getting started with the atlas-osb on Cloud Foundry

The atlas-osb installs into Cloud Foundry with a usual `cf push` command. It does not require a tile to be installed into PCF Ops Manager. These instuctions show a basic installation, configuration, and use of the broker. These have been tested on PCF 2.9. 

*NOTE* These instuctions only apply to the new template-based atlas-osb and _not_ the legacy [MongoDB Atlas Service Broker](https://github.com/mongodb/mongodb-atlas-service-broker).

## Prereq's

Typical cli/bash commands are used for this procedure along with the following tools:

* `cf` Cloud Foundry cli
* `hammer` cli (recommended)
* text editor
* MongoDB Atlas account - you need an Org apikey

Once you have your workstation ready, head over to http://cloud.mongodb.com and create an organization apikey. Step by step details on this step here: https://docs.atlas.mongodb.com/configure-api-access/#manage-programmatic-access-to-an-organization.

## Installation & Configuration

1. Pull down the latest release of the atlas-osb (recommended).

```bash
curl -OL https://github.com/jasonmimick/atlas-osb/archive/atlas-osb-latest.tar.gz
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
   "projects": {

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

Save this in a file called `keys`

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
cf set-env atlas-broker ATLAS_BROKER_TEMPLATEDIR ./samples/plans

cf set-env atlas-broker BROKER_HOST 0.0.0.0
cf set-env atlas-broker BROKER_PORT 8080

# start it up
cf start atlas-osb
```bash

Check the logs, you should see output included the loaded plan templates.

```bash
cf logs atlas-broker --recent
```

4. Register the app as a real service broker

Grab the `routes` value from `cf app atlas-osb` and use this URL in the following command.

```bash
# create the actual broker
cf create-service-broker atlas-osb admin admin <YOUR-DEPLOYMENT-URL>
```

Check out the results.

```bash
cf marketplace
cf apps
```

3. Enable service access.

Run the `cf service-access -b mongodb-atlas` command to inspect all the services now available through your Atlas broker. There are services mapping to each of the cloud providers (AWS, GCP, Azure) on which Atlas will deploy your MongoDB clusters. Here, we'll enable access to the Azure plans. Once done, you can inspect the available plans in the marketplace.

```
cf enable-service-access mongodb-atlas-azure
cf marketplace
Getting services from marketplace in org mongodb-testing / space jason as admin...
OK

service               plans                                          description                          broker
mongodb-atlas-azure   M10, M20, M30, M40, M50, M200, M60, M80, M90   Atlas cluster hosted on "AZURE"      mongodb-atlas
```

4. Create an Atlas cluster

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

