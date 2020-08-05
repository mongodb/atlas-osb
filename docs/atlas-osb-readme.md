<img align="left" width="100" height="100" 
src="https://webassets.mongodb.com/_com_assets/cms/vectors-anchor-circle-mydmar539a.svg">
atlas-osb
</img>


[![Generic badge](https://img.shields.io/badge/WORKING-PREBETA-YELLOW.svg)](https://shields.io/)

![Tanzu Application Service](https://github.com/jasonmimick/atlas-osb/workflows/Clean%20CF/badge.svg)


<img align="right" src="/docs/coming-later.svg">

Table of Contents
=====
  * [Intoduction](#introduction)
  * [Getting Started](#getting-started)
  * [Specifications](#specifications)
    * [Configuration Reference](#configuration-reference)
    * [Multiple API Key Support](#multiple-api-key-support)
    * [Atlas Plan Templates](#atlas-plan-templates)
    * [Reference Sample Basic Full Example](#reference-sample-basic-full-example)
  * [VMWare Tanzu Application Service](#vmware-tanzu-application-service)
  * [License](#license)
  * [Development](#development)


This project is a new version of the original [MongoDB Atlas Serivce Broker](https://github.com/mongodb/mongodb-atlas-service-broker) called "atlas-osb"

Atlas-osb adds the ability to define your own plans from Atlas resource templates. This new and powerful feature brings the broker to a new level of production-readiness. One simple `create-service` command is all you need to provision a complete Atlas deployment including a Project, Cluster, Database user, firewall access, and more.

The atlas-osb also adds a new level of security and configurability with the ability to deploy the broker with multiple Atlas apikeys and also integrate these keys into CredHub and Secrets.

Use the Atlas Service Broker to connect to [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) from any platform which supports the [Open Service Broker API](https://www.openservicebrokerapi.org/), such as [Kubernetes](https://kubernetes.io/) and [Pivotal Cloud Foundry](https://pivotal.io/open-service-broker).

- Provision managed MongoDB clusters on Atlas directly from your platform of choice. Includes support for all cluster configuration settings and cloud providers available on Atlas.
- Manage and scale clusters without leaving your platform.
- Create bindings to allow your applications access to clusters.

atlas-osb custom plans allow cluster administrators to manage real-world production scenarios for the MongoDB Atlas Platform within any OSB-compliant environment. Custom plan's are templates of [Atlas API](https://docs.atlas.mongodb.com/reference/api) resources.

Custom plans use the Atlas API resources and allow users to define reusable packages of related Atlas resources. These are known as "plan's". Cluster administrators can deploy these plans into Open Service Broker API marketplaces. The plans are then available through standard service broker [catalog](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#catalog-management) integration. This is available in Cloud Foundry, Kubernetes via [Service Catalog](https://github.com/kubernetes-sigs/service-catalog), and other modern containerized computing environments.

# Getting Started

## Prereq's for Cloud Foundry

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

### [How to bind and connect to a specific database](#how-to-bind-db)

You need to override the role when creating the binding like this:

```bash
 cf bind-service hello-atlas-cf wed-demo-1 -c '{"User" : {"roles" : [ { "roleName" : "readWrite", "databaseName" : "mydatabase"} ] } }'
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

### Updating a cluster

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

# Specification 

## Configuration Reference

Configuration is handled with environment variables. Logs are written to
`stdout/stderr` as appropriate and each line is in a structured JSON format.

| Variable | Default | Description |
| -------- | ------- | ----------- |
| ATLAS_BASE_URL | `https://cloud.mongodb.com` | Base URL used for Atlas API connections |
| BROKER_HOST | `127.0.0.1` | Address which the broker server listens on |
| BROKER_PORT | `4000` | Port which the broker server listens on |
| BROKER_LOG_LEVEL | `INFO` | Accepted values: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| BROKER_TLS_CERT_FILE | | Path to a certificate file to use for TLS. Leave empty to disable TLS. |
| BROKER_TLS_KEY_FILE | | Path to private key file to use for TLS. Leave empty to disable TLS. |
| PROVIDERS_WHITELIST_FILE | | Path to a JSON file containing limitations for providers and their plans. |
| BROKER_APIKEYS | | Path to file or JSON string containing credentials.
| ATLAS_BROKER_TEMPLATEDIR | | Path to folder containing plans e.g. ./samples/plans |

## Multiple API Key Support

These are the requirements for supporting multiple apikeys. Note: Sometimes we use the term "CredHub" to refer to this overall feature to support multiple keys.

1. The broker will accept a json/yaml object containing authentication credentials. The format for this will be:

```yaml
bindingName: 'Main-Creds'
broker:
  username: 'admin'
  password: 'admin' 
projects:
- id: 'first-key'
  desc: 'the first key'
  publicKey: '12345'
  privatekey: '12345'
  roles: 
  - groupId: '12345'        # TODO update to support `projectId` here
- id: 'the 2nd key'         
  publicKey: '12345'
  privateKey: '12345'
  roles: 
  - groupId: '545454'        # TODO update to support `projectId` here
orgs:
- publicKey: '12345'
  privateKey: '12345'
  id: 'the org 1 key'
  roles:
  - orgId: '3030303030'
```

Where each key conforms to the following schema defined by the [ApiKey](https://github.com/mongodb/go-client-mongodb-atlas/blob/5a4b267c469e8a4baedb1b27a1f189de1e69bfd6/mongodbatlas/api_keys.go#L36) struct.

Multi-apikey support requirements (_P#_ where _#_ is priority with 0 highest.)

1. P0 Reading keys as a string from the `BROKER_APIKEYS` environment variable.

2. P0 Reading path from the `BROKER_APIKEYS` environment variable and then loading from file.

3. P0 Reading from Cloud Foundry `VCAP_SERVICES` to support CredHub integration.

4. P1 Support reading from Kubernetes Secrets.

5. The broker will allow selection of appropriate apikey during plan provisioning by discovery of the `ApiKey` in the plan-template instance context (see below [Plans](#reference-sample-basic-full-example)).

## Atlas Plan Templates

Providing support for all the myriad of combinations of Atlas resources and arbitrary validation logic is not possible with the current broker "service"/”plan” design. atlas-osb introduces an additional way to define new “services” and “plans” which represent arbitrary Atlas resource objects (object graphs) by leveraging the declarative design of the Atlas API and JSON templates.

Most users and most apps need the following resources at minimum for typical usage:

* 1 Atlas Project to contain everything
* 1 Standard 3-node replica set for your database
* 1 IP-Whitelist so that your app can connect
* 1 DB credential, again, so your app can connect

We can model the above with the following set of templates:

### Reference Sample Basic Full Example

```yaml
name: basic-plan
description: This is the `Basic Plan` template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection.
free: true
apiKey: {{ mustToJson (index .credentials.Orgs (default "" .org_id)) }}
project:
  name: {{ .instance_name }}
  desc: Created from a template
  orgId: {{ .org_id }}
cluster:
  name: {{ .instance_name }}
  providerBackupEnabled: {{ default "true" .backups }}
  providerSettings:
    providerName: {{ default "AWS" .provider }}
    instanceSizeName: {{ default "M20" .instance_size }}
    regionName: {{ default "US_EAST_1" .region }}
  labels:
    - key: Infrastructure Tool
      value: MongoDB Atlas Service Broker
databaseUsers:
- username: {{ default "test-user" .username }}
  password: {{ default "test-password" .password }}
  databaseName: {{ default "admin" .auth_db }}
  roles:
  - roleName: {{ default "readWrite" .role }}
    databaseName: {{ default "default" .role_db }}
ipWhitelists:
- ipAddress: "0.0.0.0/1"
  comment: "everything"
```
### Requirements

1. P0 Support loading Plan templates from json or yaml files mounted into the broker runtime at deployment-time.
2. The broker should support reading plan templates directly from Linux environment variables. 
    1. P0 Support a default directory to load templates from `ATLAS_BROKER_TEMPLATEDIR`

    ATLAS_BROKER_TEMPLATEDIR = "/templates"
    UPS - user provided service to mount files into /templates
    K8s - mount configmaps as files, etc..., Docker, etc..
3. Include loaded templates with INFO level and also full json payload logging to broker logs (to help debugging).
4. Always reload templates on startup (restart to deploy new templates)
5. Template shoud be standard go-templates and support typical values/variable style replacements.
6. Users should be able to specify template parameters during service provisioning or update.
7. Provide a set of common resource yaml/json samples and templates.
8. Allow reading apikeys and group/org ids from the "CredHub" multi-apikey support
    1. Make a template variable called `Credentials` available which contains all the apikeys
9.   Allow reading apikey from a yaml/json file with a resource definition.
10. :construction: Support a `--dry-run` flag whenever processing a provision, update, or delete operation on a custom plan. Default is `false`. Include pre & post template processing in logging output. (Not supported yet.)

### Plan Functional Design

Each Plan instance is managed through the OSB provision, bind, unbind, and deprovision operations. 
In this section we describe the relationship between a Plan's OSB operations and how the broker translates them to Atlas Client API calls.

Each plan will have at least one apikey associated with it.

:construction:

#### Provisioning & Deprovisioning

When the broker gets a call to provision a plan, it will iterate through the various Atlas resources in the plan can call the corresponding service `Create` method. Similarily, when deprovisioning, the broker will delegate calls to the Atlas Go-client corresponding service `Delete` function.

Plans are loaded at startup and first validated before being made available in the Marketplace.

1. read plans from disk
2. do a first-pass template parse with empty context (substitute "" for everything in template)
   * Allow for default values? 
   * Provide a way to test the plan actually works (dry-run)
3. parse the result into yaml, get static metadata like plan name, description, instance size, whatever
4. on provision, do a parse with full context - this is the final plan spec
   * :construction: Allow for dry-run at this step too. 

#### Managing State

This section describes how the state of plan definitions and service instance metadata will be stored. 

atlas-osb introduces the use of MongoDB Realm applications to manage the state of your deployed service instances. This happens automatically, directly through the Atlas and Realm APIs.

The atlas-osb will create special project called "Atlas Service Broker Maintenance" for each Atlas Organization. Here,  the broker will create a Realm App called "Broker State" :constuction: and use Realm Values to store reference data for the serivce instances you have deployed across your environments. This approach has several advantages:

* zero-footprint on OSB-marketplace environment
* single location for all serivce instance metadata
* follows the Atlas organization boundaries, allows support for multi-tenancy scenarios
* automaticalluy encrypted via Realm 

See [Realm Values & Secrets](https://docs.mongodb.com/realm/values-and-secrets/)

### Bind & Unbind

The OSB bind function is used to provision a new database user credential and connection information for an application using MongoDB. This usually happens when an app is deployed into a new environment. To support this, the broker will create new Atlas resources for the binding and return the connection information appropriately. 

:construction: broker needs refactor to use Plan user/ipwhitelist.

When the broker creates a binding, it will translate the Connection Details for the given cluster into the OSB Binding structure. 

[Connection Details](https://github.com/jasonmimick/atlas-osb/blob/a503c88b66c9df15f8620c7f072826ba13ca3dd3/pkg/broker/binding_operations.go#L16) types.
    
The format for the JSON available for binding in `VCAP_SERVICES` is:

```json
{ 'connectionString': 'mongodb+srv://uuuuuuuuu:xxxxxxxx@chewy-123.6bikq.mongodb.net/admin',
  'password': 'xxxxxxxxxx',
  'uri': 'mongodb+srv://chewy-123.6bikq.mongodb.net',
  'username': 'uuuuuuuuuu',
  'datbase': 'admin'}
```

Please see the [test/hello-atlas-cf](test/hello-atlas-cf) sample app to see details on the binding information available to apps.

#### Overriding the database for all bindings

Certain customers may wish to control the exact name of the database to which apps using Atlas services can use. This is controlled by inserting the database name into the connection string (as the last forward-slash piece before the query string) which is constructed during a call to the brokers Bind function.

In general, we do not recommend using this feature. However it will be released as "Deprecated" in the Beta release for the broker. Customers are encouraged to refactor their dependencies on fixing this database name.

###### Default Bind Roles
For example, to set the default role for any database users created via the bind() OSB operation, add these to the `settings` for your plan.

```yaml
name: override-bind-db-plan
description: This is an extension of the `Basic Plan` template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection. But it added the ability to override the bind db.
free: true
apiKey: {{ mustToJson (index .credentials.Orgs (default "" .org_id)) }}
settings:
  overrideBindDB: "products"
  overrideBindDBRole: "readWrite" 
project:
  name: {{ .instance_name }}
  desc: Created from a template
...
```

This sets gives `readWrite` on the `products` database for each new user created via bind().

### Plans and Atlas Resource Types 

The following types are supported for loading from multiple or a single yaml or json objects.

Note these types are derived from the following specifications:

* [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker)
* [MongoDB Atlas API](https://docs.atlas.mongodb.com/api/)
* [MongoDB Atlas Go client](https://github.com/mongodb/go-client-mongodb-atlas)

Each "plan" is loaded from a single json/yaml object describing the resources managed by the plan. Plans can contain any number of various MongoDB Atlas resources, but require an Atlas Project. Since each Atlas resource is always associated with an Atlas Project, each Plan must have a project.
The other resources are optional.

The `Plan` and `Binding` types are new structures to implement this new feature, while the other types are used directly from the [MongoDB Atlas Go client](https://github.com/mongodb/go-client-mongodb-atlas).

The officially supported Atlas Resources are:

* [Project](#project)
* [Cluster](#cluster)
* [DatabaseUser](#databaseuser)
* [ProjectIPWhitelist](#projectipwhitelist)

#### Plan

This represents an Open Service Broker [plan](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#service-plan-object) (NOTE:  https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#service-offering-object)

```go
// Plan represents a set of MongoDB Atlas resources
type Plan struct {
    Version         string                              `json:"version,omitempty"`
    Name            string                              `json:"name,omitempty"`
    Description     string                              `json:"description,omitempty"`
    ApiKey          *mongodbatlas.ApiKey                `json:"apiKey,omitempty"`
    Project         *Project                            `json:"project,omitempty"`
    Clusters        []*mongodbatlas.Cluster             `json:"clusters,omitempty"`
    DatabaseUsers   []*mongodbatlas.DatabaseUser        `json:"databaseUsers,omitempty" yaml:"databaseUsers,omitempty"`
    IPWhitelists    []*mongodbatlas.ProjectIPWhitelist  `json:"ipWhitelists,omitempty" yaml:"ipWhitelists,omitempty"`
    Bindings        []*Bindings                   // READ ONLY! Populated by bind()
    Settings            map[string]string                 `json:"settings,omitempty"`
}
```

The remaining resource type definitions are taken directly from the Atlas Go Client, and therefore subject to change per that project.

* #### Project

An Atlas [project](https://docs.atlas.mongodb.com/reference/api/projects/) (NOTE:  https://docs.atlas.mongodb.com/reference/api/projects/). Requires an organization id or valid context from apikey.

[Project](https://github.com/mongodb/go-client-mongodb-atlas/blob/a5ca32cb21bbad57486c011c3f51ec853b76c123/mongodbatlas/projects.go#L45) type.

* #### Cluster

An Atlas [cluster](https://docs.atlas.mongodb.com/reference/api/clusters-create-one/#example-request) (NOTE:  https://docs.atlas.mongodb.com/reference/api/clusters-create-one/#example-request).

[Cluster](https://github.com/mongodb/go-client-mongodb-atlas/blob/master/mongodbatlas/clusters.go#L93)

TODO: MARK WHICH FIELDS ARE READ-ONLY? ie. users need to understand what can be in the template

* #### Database User

[Database_Users](https://github.com/mongodb/go-client-mongodb-atlas/blob/master/mongodbatlas/database_users.go)

* #### Project IP Whitelist

[Project_IP_Whitelist](https://github.com/mongodb/go-client-mongodb-atlas/blob/master/mongodbatlas/project_ip_whitelist.go)


# VMWare Tanzu Application Service
:construction:

## License

See [LICENSE](LICENSE). Licenses for all third-party dependencies are included in [notices](notices).

## Development

Information regarding development, testing, and releasing can be found in the [development documentation](dev).
