name: multi-region-us
description: "This is the `Basic Plan` template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection."
free: true
apiKey: {{ mustToJson (index .Credentials.Orgs (default "" .org_id)) }}
project:
  name: {{ .instance_name }}
  desc: Created from a template
  orgId: {{ .org_id }}
cluster:
  name: {{ .instance_name }}
  clusterType: "REPLICASET"
  providerSettings:
    providerName: {{ default "AZURE" .provider }}
    instanceSizeName: {{ default "M30" .instance_size }}
  replicationSpecs:
  - numShards: 1
    zoneName: "US Zone"
    regionsConfig:
      US_CENTRAL:
        analyticsNodes: 0
        electableNodes: 1
        priority: 6
        readOnlyNodes: 0
      US_EAST_2:
        analyticsNodes: 0
        electableNodes: 2
        priority: 7
        readOnlyNodes: 0
      US_WEST:
        analyticsNodes: 0
        electableNodes: 2
        priority: 5
        readOnlyNodes: 0
databaseUsers:
- username: {{ default "test-user" .username }}
  password: {{ default "test-password" .password }}
  databaseName: {{ default "admin" .auth_db }}
  roles:
  - roleName: {{ default "readWrite" .role }}
    databaseName: {{ default "test" .role_db }}
ipWhitelists:
- ipAddress: "0.0.0.0/1"
  comment: "everything"
- ipAddress: "128.0.0.0/1"
  comment: "everything"
