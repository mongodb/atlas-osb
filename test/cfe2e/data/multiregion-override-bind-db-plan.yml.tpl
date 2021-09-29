name: multiregion-override-bind-db-plan
description: This is an extension of the `Basic Plan` to a multi-region database cluster template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection. But it added the ability to override the bind db.
free: true
apiKey: {{ keyByAlias .credentials "testKey" }}
settings:
  overrideBindDB: "OriginalMongoDBTileForPCFDBName"
  overrideBindDBRole: "readWrite"
  overrideAtlasUserRoles: [GROUP_OWNER]
project:
  name: {{ .instance_name }}
  desc: Created from a multiregion template
cluster:
  name: {{ .instance_name }}
  providerBackupEnabled: {{ default "true" .backups }}
  clusterType: "REPLICASET"
  providerSettings:
    providerName: {{ default "AZURE" .provider }}
    instanceSizeName: {{ default "M10" .instance_size }}
  replicationSpecs:
  - numShards: 1
    zoneName: "US Zone"
    regionsConfig:
      NORWAY_EAST:
        analyticsNodes: 0
        electableNodes: 1
        priority: 6
        readOnlyNodes: 0
      GERMANY_NORTH:
        analyticsNodes: 0
        electableNodes: 2
        priority: 7
        readOnlyNodes: 0
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
ipAccessLists:
- ipAddress: "0.0.0.0/1"
  comment: "everything"
- ipAddress: "128.0.0.0/1"
  comment: "everything"
