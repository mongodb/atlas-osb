atlas-osb Beta Launch
=====

:construction:

## Why did we start the atlas-osb project?

MongoDB customers using VMWare Tanzu Application Service (aka Pivotal PAS, Cloud Foundry) need a way to use MongoDB Atlas. MongoDB has also deprecated it's PCF "Tile", so we started this project to help customers migrate to Atlas. The original Atlas Open Service broker did not satisfy the requirements for customers, so we started this project to fix that.

## What is the atlas-osb?

atlas-osb is a fork of the original Atlas Open Service Broker. 

This project will be released as a new version of the Atlas Open Service Broker.

Branded as a "*New and Improved Atlas OSB V2*", atlas-osb is not really "technically" compatible but it is functionally equivallent (everything you could do with the broker V1 you can do with V2). It is also way more feature packed. You can manage database users, network-peering, projects, _connections to MongoDB databases_, and more directly with the new broker. This is all possible through the new extensible Atlas Plan Templates which power atlas-osb. 

## Beta Launch Countdown

LAUNCH DATE: TUESDAY, AUGUST 25, 2020

## Beta Launch Release Process

We will TAG releases in the repo for milestones, and publish these for customers.


## More notes about atlas-osb

The Atlas Plan Templates which power atlas-osb are simple declarative yaml files which describe *exactly* the MongoDB Atlas resources you wish to provision for a given Open Service Broker Marketplace plan offering. Cluster administrator now have the ability to build customize "menus" of highly-complex solution offerings of MongoDB Atlas functionality. For example, a typlical mainframe offloading scenario would require a standard set of Atlas resources for a production deployment:

1. MongoDB Atlas Project
2. Cluster
3. Database User(s)
4. IP-Whitelist
5. Network Peering
6. Backup Schedules
7. Event notification
8. Connection Strings for apps

All of these resources can be packaged up together into one Plan definition. Users can then deploy the entire solution with the click of a button.

You can read more on the main [README.md](../README.md).



## Technical details

On the code side, we forked the original repo and then added a bunch of stuff. 
Fixed bugs, added few more odd-ball features, but then stripped the code base down to just support the new the `[dynamicplans.Plan](/pkg/broker/dynamicplans/plan.go)` type and only support `[credentials.Credentials](/pkg/broker/credentials/credentials.go)` for authentication and authoriazation. Note, we separated the local broker HTTP Digest auth and the Atlas API keys auth, it's not a passthrough anylonger.

Developer/Support Notes

- <add note about new realm api, how does auth token work>
- <add note about realm value and statestorage>
  
