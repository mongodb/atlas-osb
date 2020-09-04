Minimum Product Requirements for the Atlas Service Broker
===

* Create a cluster
  * Single Region
  * Multiple Regions
* Bind
  * customer roles? db?
  * whitelist ip?
* Pause Cluster
* Scale Cluster tier, e.g M20-M30 (not replset to sharded cluster)
* Configure Backups
  * Ensure enable/disable is functional
* Install broker into VMware/Cloud Foundry
* Ensure apps running in VMware PAS (Pivotal Application Server) can
  * Create and delete database access credentials through service broker binding
  * Connect to and access MongoDB clusters
