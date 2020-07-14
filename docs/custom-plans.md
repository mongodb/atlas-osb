# Introduction 

Atlas Service Broker (broker) Custom Plans with Resource Templates (CPT's) allow cluster administrators to manage real-world production scenarios on the MongoDB Atlas Platform. CPT plan's are templates of Atlas resources.

Without CTP's the broker only supports very basic clusters, basic db-user through bind(), and only a single apikey. These limitations severely hinder adoption and do not support typical enterprise customer Atlas use cases.

CTP's use the existing Atlas API resources and allow users to define a reusable package of related Atlas resources known as a "plan". Cluster administrators can define and deploy these "marketplace plans" into an ASB instance. The plans are then available throug standard service broker [catalog](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#catalog-management) integration. This is available in Cloud Foundry, Kubernetes via [Service Catalog](https://github.com/kubernetes-sigs/service-catalog), and other modern containerized computing environments.

# Problem

The broker implements the [osbapi](https://github.com/openservicebrokerapi) and, so, builds a catalog consisting of a set of "services" each with “plans”. By default, the broker creates a service for each Atlas cloud provider (aws, gcp, azure) and then for each of those a plan for each Atlas cluster size (M10, M20, …).  

The broker also only supports a single api key, meaning users are limited to using the broker with existing Atlas projects (and database users, an ip whitelists, etc…).

Customers have requested the broker provide functionality to manage the additional Atlas resources required to actually deploy and manage production applications using Atlas data services. This means, when users request a "cluster" what they really want is some set of Atlas resources - for example, a project, with 1 replica set, with 1 db-user limited to, say, `readWrite` on the `myapp.mydb` namespace. 

# Solution

The broker will support 3 modes of "opertation":

1. Default 
    1. deploy broker without any ApiKey, and just use key passed through from broker client calls
       (not applicable to cloud foundry)
    2. deploy with 1 apikey

2. Multiple ApiKey support New feature below - 

3. Plan Templates with ApiKeys

* This feature allows administrators to build and market customized Atlas "plans" consisting of Atlas Projects, Clusters, Users, etc.
* The Atlas resources managed through this feature are running in the MongoDB Atlas Cloud. You may incure cost for said use.
* The ultimate "source of truth" for all Atlas resources managed through this feature is Atlas itself. 

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

5. P0 The broker will support auto-generating plans from projects.
    The broker will support a feature to dynmically create a set of plans for each project discovered through the set of loaded apikeys. This feature is designed to make it simple for a typical use-case where you don't need advanced resource management and instead need a simple way to let users create clusters in a fixed set of Atlas projects.
   1. An environment flag `BROKER_ENABLE_AUTOPLANSFROMPROJECTS`, Default to false.
   2. Requires restart to detect change, so we can turn off that feature (the feature to generate the plans)
   3. Plan names are structured as strings such as `MyProjectA-M10` where the format is `<PROJECT_NAME>_<CLUSTER_SIZE>`

6. The broker will allow selection of appropriate apikey during plan provisioning as follows:
   1. Key detected from auto-generated plan. See [plan-metadata](#plan-metadata). 
   2. The `ApiKey` in template context see [multi-resource](#multiple-atlas-resource-support) support below.
   3. The "create-service" parameter `apikey` or just `key`

7. Multiple apikey support can be turned off and on.
   

## Atlas Plan Templates

Providing support for all the myriad of combinations of Atlas resources and arbitrary validation logic is not possible with the current broker "service"/”plan” design. We propose an additional way to define new “services” and “plans” which represent arbitrary Atlas resource objects (object graphs) by leveraging the declarative design of the Atlas API and JSON templates.

Most users and most apps need the following resources at minimum for typical usage:

* 1 Atlas Project to contain everything
* 1 Standard 3-node replica set for your database
* 1 IP-Whitelist so that your app can connect
* 1 DB credential, again, so your app can connect

We can model the above with the following set of templates:

### Reference Sample Basic Full Example

```yaml
name: MyPlan
description: "This is a sample plan for a project, cluster, database user, and secure connection."
apiKey:
  name: {{ .Context.Keys(.SomeValueMyClusterAdminTellMe) }}  // lookup into CredHub keys "dict"
project:
  name: {{ .ProjectName }}
``

yaml
name: MyPlan
description: "This is a sample plan for a project, cluster, database user, and secure connection."
apiKey:
  name: {{ .ProjectName }}-ApiKey
  privateKey: {{ .PrivateKey }}
  publicKey: {{ .PublicKey }}
  orgId: {{ .OrgId }}
project:
  name: {{ .ProjectName }}
  orgId: {{ .OrgId }}
clusters:
- name: {{ .ProjectName }}-Cluster
  groupId: {{  .Plan.Project.ID }}
databaseUsers:
- username: "test-user"
  password: "test-password"
  databaseName: "admin"
  groupId: {{  .Plan.Project.ID }}
  roles:
  - roleName: "readWrite"
    collectionName: {{ .CollectionName }}
    databaseName: {{ .DatabaseName }}
ipWhitelists:
- ipAddress: "192.168.1.1"
  comment: "test-ip"
  groupId: {{  .Plan.Project.ID }}
```

  
### Requirements

1. P0 Support loading Plan templates from json or yaml files mounted into the broker runtime at deployment-time.
2. The broker should support reading plan templates directly from Linux environment variables. 
    1. P0 Support a default directory to load templates from `ATLAS_BROKER_TEMPLATEDIR`

    ATLAS_BROKER_TEMPLATEDIR = "/templates"
    UPS - user provided service to mount files into /templates
    K8s - mount configmaps as files, etc..., Docker, etc..

    2. P2 Support uri-format paths, e.g. [http://my-stuff/my-atlas-template.json](http://my-stuff/my-atlas-template.json) or github
3. Include loaded templates with INFO level and also full json payload logging to broker logs (to help debugging).
4. Always reload templates on startup (restart to deploy new templates)
5. Template shoud be standard go-templates and support typical values/variable style replacements.
6. Users should be able to specify template parameters during service provisioning or update.
8. Provide a set of common resource yaml/json samples and templates.
9. Allow reading apikeys and group/org ids from the "CredHub" multi-apikey support
    1. Make a template variable called `Credentials` available which contains all the apikeys
10.  Allow reading apikey from a yaml/json file with a resource definition.
11.  Support a `--dry-run` flag whenever processing a provision, update, or delete operation on a custom plan. Default is `false`. Include pre & post template processing in logging output.

  
#### Implementation Notes

* We can use [user provided services](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html) for the cf broker deployment to mount the plans into the broker runtime.
* For k8s we can just mount configmap
  
#### Open Questions

1. How to deal with osb bind call? Need to map the DB User and IP Whitelist resource (maybe others) to the bind [Credential](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#types-of-binding) concept


## Atlas Service Broker Plan Template Specifications

### Introduction

The following types are supported for loading from multiple or a single yaml or json objects.

Note these types are derived from the following specifications:

* [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker)
* [MongoDB Atlas API](https://docs.atlas.mongodb.com/api/)
* [MongoDB Atlas Go client](https://github.com/mongodb/go-client-mongodb-atlas)

Each "plan" is loaded from a single json/yaml object describing the resources managed by the plan. Plans can contain any number of various MongoDB Atlas resources, but require an Atlas Project. Since each Atlas resource is always associated with an Atlas Project, each Plan must have a project.
The other resources are optional.

### Plans and Atlas Resource Types 

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
    DefaultBindingRoles  *[]mongodbatlas.Role           `json:"defaultBindingRoles"`
    Bindings        []*Bindings                   // READ ONLY! Populated by bind()
}

type Binding struct {
  //Binding info
}
```

###### Default Bind Roles
For example, to set the default role for any database users created via the bind() OSB operation,

```yaml
# Your own plan.yaml
name: MyPlan
description: "This is a sample plan for a project, cluster, database user, and secure connection."
apiKeys:
- name: {{ .ProjectName }}-ApiKey
  privateKey: {{ .PrivateKey }}
  publicKey: {{ .PublicKey }}
  orgId: {{ .OrgId }}
project:
  name: {{ .ProjectName }}
  orgId: {{ .OrgId }}
clusters:
- name: {{ .ProjectName }}-Cluster
  groupId: {{  .Plan.Project.ID }}
settings:
  defaultBindingRoles:
  - name: "readAnyDatabase"
  - name: "write"
    databaseName: "myAppDB"
    collectionName: "myAppCollection"
  - name: "clusterMonitor"
```

This sets 3 roles for every new user created via bind().

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


### Plan Functional Design

Each Plan instance is managed through the OSB provision, bind, unbind, and deprovision operations. 
In this section we describe the relationship between a Plan's OSB operations and how the broker translates them to Atlas Client API calls.

Each plan will have at least one apikey associated with it. This is discovered by the following rules during the provision/update/deprovision or bind/unbind calls:

1. ApiKey resource directly in plan can be either inline ApiKey or reference to Multi-Key/CredHub key already deployed with the broker. For references to CredHub apikey's templates will support the feature to lookup the CredHub keys by name in plans. For example, &#x1F4DD; UPDATE THIS WITH IMPLEMENTATION DETAILS

   ```yaml
   apikey:
     name: {{ .Broker.Credentials[.SomeApiKeyName] }}
   ```
   where `.Broker` is a contextual variable available to template designers, which has a Map[string][ApiKey] called `Credentials`. This template would use the value of the `SomeApiKeyName` instance property (from values.yaml) to as the key into the Credentials map which gets loaded from the [Multi-key support](#multiple-api-key-support).

2. Try to load an `ApiKey` struct from the instance property called `apikey`, for example:
   ```bash
   cf create-service myplan myinstance -c { "apikey" : { "publicApiKey" : ...}}
   ```

#### Provisioning & Deprovisioning

When the broker gets a call to provision a plan, it will iterate through the various Atlas resources in the plan can call the corresponding service `Create` method. Similarily, when deprovisioning, the broker will delegate calls to the Atlas Go-client corresponding service `Delete` function.

[TODO] How to handle existing resources?
[TODO] How to handle say a cluster in a project but not in a plan?

Plans are loaded at startup and first validated before being made available in the Marketplace.

1. read plans from disk
2. do a first-pass template parse with empty context (substitute "" for everything in template)
   * Allow for default values? 
   * Provide a way to test the plan actually works (dry-run)
3. parse the result into yaml, get static metadata like plan name, description, instance size, whatever
4. on provision, do a parse with full context - this is the final plan spec
   * Allow for dry-run at this step too. 


##### Managing State

This section describes how the state of plan definitions and service instance metadata will be stored. 

We need some kind of storage for each plan instance, this can be either local to where the broker is running or we can use Atlas itself. Here are the options:

A) Store plan instance metadata in some s3-style bucket. Simple way to fetch and store without any local requirements. 
B) If on kubernetes, then store in simple ConfigMap
C) If on cloud foundry, install Minio or other s3-compatible



### Bind & Unbind

The OSB bind function is used to provision a new database user credential and connection information for an application using MongoDB. This usually happens when an app is deployed into a new environment. To support this, the broker will create new Atlas resources for the binding and return the connection information appropriately. 


The `bind()` call maps to `DatabaseUserService.Create` then `ProjectIPWhiteListService.Create`, and 
`unbind()` maps to `ProjectIPWhiteListService.Delete` then `DatabaseUserService.Delete` (note order change).

When the broker creates a binding, it will translate the Connection Details for the given cluster into the OSB Binding structure. 

[Connection Details](https://github.com/jasonmimick/atlas-osb/blob/a503c88b66c9df15f8620c7f072826ba13ca3dd3/pkg/broker/binding_operations.go#L16) types.
    
The format for the JSON available for binding in `VCAP_SERVICES` is:

```json
{ 'connectionString': 'mongodb+srv://uuuuuuuuu:xxxxxxxx@chewy-123.6bikq.mongodb.net/admin',
  'password': 'xxxxxxxxxx',
  'uri': 'mongodb+srv://chewy-123.6bikq.mongodb.net',
  'username': 'uuuuuuuuuu'}
```

Please see the [test/hello-atlas-cf](test/hello-atlas-cf) sample app to see details on the binding information available to apps.

_*FUTURE SPRINT PROPOSAL*_

The binding feature will be enchanced to support different connection string format.
The proposal is to add this at the `Plan` level, adding a template for the binding.

For example,

```yaml
bindingParameters:
  connectionString:
    source: standardSrv
    format: $proto://$user:$pass@$host/$dbName?authSource=$authSource&whatever=else
```

If the `format` is empty, just take the provided source, defaulting to pure standard or standardSrv returned from Atlas.

*TODO* design and document the parameters such as `$protocol` for the binding format.



