The Atlas service broker is typically deployed with a  
single MongoDB Atlas programatic apikey. This apikey is associated  
with an Atlas project. If you need to manage Atlas clusters which span more than one Atlas project  
then you have the following options:

```json
{
   "broker": {
      "username": "admin",
      "password": "admin"
   },
   "orgs": {
      "5ea6d37fff972e2d63e7e520": {
         "public_key": "CTWZPIFJ",
         "api_key": "d7cf7772-fe56-4033-9a8d-1825432d51ef",
         "display_name": "testOrg"
      }
   },
   "projects": {
      "5ea6d37fff972e2d63e7e520": {
         "public_key": "CTWZPIFJ",
         "api_key": "d7cf7772-fe56-4033-9a8d-1825432d51ef",
         "display_name": "test"
      }
   }
}
```

1. Register multiple broker instances. This is a lightweight way to support multiple projects. Simply create an Atlas broker instance in your environment named for and along with an apikey for each project.

2. Customize the plans offered by your broker instance. See [Adding apikeys to plans](#adding-apikey-to-plan).

## Adding apikeys to plans

When the Atlas service broker launches it first pulls down a complete catalog from the Atlas cloud which contains all the various offerings (one for each instance size and cloud provider, e.g. mongodb-atlas-aws-M40). This list of plans is then filtered through the broker whitelist file.

When the broker is configured to use multiple apikeys the plans will then be enhanced as follows:

* Full catalog pulled from Atlas cloud
* Catalog filtered through broker whitelist (only provider and sizes in the whitelist are used)
* For each provider (AWS, Azure, GCP)
  * For each cluster size (M10,M20,...)
    * For each Atlas organization apikey
      * For each project in the Atlas 
        * Add a plan (provider,cluster-size,org,project) 
    * For each Atlas project apikey
        * Add a plan (provider,cluster-size,org,project) 
      
#### Plan names

Plans are uniquely defined by building a list of instance sizes filtered through and a project name ("display name" from json-file). For each such pair and each Atlas project the broker will then create a `ServicePlan`.

Plans are named according to the following convention:
<size>-<project>

Add a plan, e.g. "mongodb-atlas-azure-M10-MyProject" (The display name is used here, see below)


Follow these steps to add a new Atlas project & apikey:

If you haven't yet deployed a Cloud Foundry app running the actual broker or created an actual cf service-broker yet, see the [Getting started](https://github.com/jasonmimick/atlas-osb/wiki/Getting-started-with-the-Atlas-Service-Broker-on-Cloud-Foundry) guide for installation of the broker. 

*for example*
```bash
cf push atlas-broker --docker-image quay.io/mongodb/mongodb-atlas-service-broker:latest --no-start
cf set-env atlas-broker BROKER_HOST 0.0.0.0
cf set-env atlas-broker BROKER_PORT 8080
cf set-env atlas-broker BROKER_ENABLE_AUTOPLANSFROMPROJECTS true
cf create-service-broker atlas-broker '<broker.username>' '<broker.password>' http://atlas-broker.apps.CF_DOMAIN
```

1. Create atlas project & apikey at cloud.mongodb.com and configure which IP addresses can access your project (Network Access > IP Whitelist).

2. Create credhub service instance which contains a JSON document which describes the Atlas projects and apikey you wish to use with a given instance of the Atlas Service Broker.

Copy the following sample JSON into a file, and save the file. For example, `my-custom-atlas-plans.json`

```json
{
  "broker": {                           
      "username": "admin",
      "password": "admin"
  },
  "5ea6d37fff972e2d63e7e520": {         
    "public_key": "123456",
    "api_key": "xxxxxxx-4444-4444-bbdd-bbbbbbbbbbbb",
    "display_name": "test"              
  },
  "<ATLAS_PROJECT_ID>": {
    "public_key": "<ATLAS_PUBLIC_APIKEY>",
    "private_key": "<ATLAS_PRIVATE_APIKEY>",
    "display_name": "Blue Team Dev"     
  }
}
```

The `broker` key holds the HTTP Digest auth credentials for the broker, but these will not work for Atlas api calls. Instead for each Atlas project id in the credhub service instance the broker will create a special plan for each Atlas cluster size allowed (through the whitelist file). 

3. Then, you can refer to this file when creating an instance of the CredHub service. This example creates a CredHub service with the `credhub` `default` plan called `atlas-credhub`. The contents of this credential are read from the file created in step 2a. 

```bash
cf create-service credhub default atlas-credhub -c ./my-custom-atlas-plans.json
```

4. Bind this credhub service instance to the broker

```bash
cf bind-service atlas-broker atlas-credhub --binding-name atlas-credhub
```

cf bind-service atlas-broker atlas-credhub
```
>>>>>>> 6a97f26ea04af84d2e5ce5dfc72f31753f7f251e:Multiple-Atlas-projects-per-broker.md

5. When you call create-service, pass the plan name, for example:

```bash
cf create-service my-atlas M10-atlas-project-x-display-name
```

Now you're ready to 



(see https://github.com/10gen/ops-manager-kubernetes/blob/63e72f57a99e9263ab7663b7fb43e5a30fc1bd64/pkg/controller/operator/groupshelper.go#L14))


>>>>>>> 6a97f26ea04af84d2e5ce5dfc72f31753f7f251e:Multiple-Atlas-projects-per-broker.md
