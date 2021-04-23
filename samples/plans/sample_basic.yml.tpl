# plan ID used when creating services
# required
name: basic-plan
# plan description for Service Catalog
description: This is the `Basic Plan` template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection.
# whether plan should be listed as free or paid (default: false)
free: true
# override apiKey for this plan
# optional; project.orgId can be used instead (see below)
# .credentials is a builtin dictionary provided by the Broker
# keyByAlias is a builtin helper provided by the Broker to select from .credentials by arbitrary name
apiKey: {{ keyByAlias .credentials "testKey" }}

# Atlas Project definition
# https://docs.atlas.mongodb.com/reference/api/project-create-one/#request-body-parameters
# required
project:
  # .instance_name is part of Platform context exposed by the Broker
  name: {{ .instance_name }}
  # orgId to use if no apiKey is provided (see above) - can be hardcoded or exposed to the user
  #orgId: {{ .orgId }}

# Atlas Cluster definition
# https://docs.atlas.mongodb.com/reference/api/clusters-create-one/#request-body-parameters
# required
cluster:
  name: {{ .instance_name }}
  # default is a builtin helper for substituting defaults instead of nil-values & empty strings
  providerBackupEnabled: {{ default "true" .backups }}
  providerSettings:
    providerName: {{ default "AWS" .provider }}
    instanceSizeName: {{ default "M10" .instance_size }}
    regionName: {{ default "US_EAST_1" .region }}
  labels:
    - key: Infrastructure Tool
      value: MongoDB Atlas Service Broker

# Atlas DatabaseUser definitions to create during provision
# https://docs.atlas.mongodb.com/reference/api/database-users-create-a-user/#request-body-parameters
# optional
databaseUsers:
- username: {{ default "test-user" .username }}
  password: {{ default "test-password" .password }}
  databaseName: {{ default "admin" .auth_db }}
  roles:
  - roleName: {{ default "readWrite" .role }}
    databaseName: {{ default "default" .role_db }}

# Atlas IP Access List definitions to create during provision
# https://docs.atlas.mongodb.com/reference/api/ip-access-list/add-entries-to-access-list/#request-body-parameters
# optional
ipAccessLists:
- ipAddress: "0.0.0.0/1"
  comment: "everything"
- ipAddress: "128.0.0.0/1"
  comment: "everything"

#privateEndpoints:
#  AZURE:
#    US_WEST_2:
#      endpoints:
#      - subscriptionID: {{ .subscriptionID }}
#        resourceGroup: rg-test
#        virtualNetworkName: test-subnet
#        subnetName: default
#        endpointName: {{ .instance_name }}