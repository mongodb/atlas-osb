DRAFT - IN - PROGRESS - 
1. Installing the service broker app

By default, the broker will only listen on localhost (127.0.0.1) and port 4000.
Use the `set-env` command to update this for Cloud Foundry defaults (port 8080).

```
 cf push mongodb-atlas --docker-image quay.io/mongodb/mongodb-atlas-service-broker:latest --no-start
 cf set-env mongodb-atlas BROKER_HOST 0.0.0.0
 cf set-env mongodb-atlas BROKER_PORT 8080
 cf start mongodb-atlas
```
2. Register the app as a real service broker

For this step, you need a MongoDB Atlas apikey. Create one at http://cloud.mongodb.com.
Create an apikey and then note the PUBLIC_KEY, PRIVATE_KEY. You will also need the PROJECT_ID of an Atlas project to host the clusters deployed by this registration of the broker. Finally, you should retrieve ATLAS_APP_URL

```
 cf app mongodb-atlas
```

Use these to register an instance of the broker to use these credentials. 

```
 cf create-service-broker mongodb-atlas '<PUBLIC_KEY>@<PROJECT_ID>' '<PRIVATE_KEY>' <ATLAS_APP_URL>
```
_NOTE You can have multiple brokers registered all pointing back to the same app deployed in step 1. In that case, you should make sure that the broker's service planes do not overlap, e.g. --space-scoped the command makes the broker's service plans only visible within the targeted space. If you use --space-scoped you can skip step 3._

```
 cf create-service-broker mongodb-atlas '<PUBLIC_KEY>@<PROJECT_ID>' '<PRIVATE_KEY>' <ATLAS_APP_URL> --space-scoped
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

```
cf create-service mongodb-atlas-azure M10 cf-big-toe -c '{"cluster":  {"providerSettings":  {"regionName": "US_EAST_2"} } }'
Creating service instance cf-big-toe in org mongodb-testing / space jason as admin...
OK

Create in progress. Use 'cf services' or 'cf service cf-big-toe' to check operation status.
```
```
➜  atlas-osb git:(master) ✗ cf service cf-big-toe
Showing info of service cf-big-toe in org mongodb-testing / space jason as admin...

name:             cf-big-toe
service:          mongodb-atlas-azure
tags:             
plan:             M10
description:      Atlas cluster hosted on "AZURE"
documentation:    
dashboard:        https://cloud.mongodb.com/v2/5eb5605b9048047417d7faf1#clusters/detail/efb648fc-f4af-4d82-a808
service broker:   mongodb-atlas

Showing status of the last operation from service cf-big-toe...

status:    create in progress
message:   
started:   2020-05-12T20:36:22Z
updated:   2020-05-12T20:36:23Z

There are no bound apps for this service.

Upgrades are not supported by this broker.
```

5. Push your app

6. Bind app

# # Advanced

* Customized plans provider-whitelist