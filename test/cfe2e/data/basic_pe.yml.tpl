name: basic-pe-plan
description: For privateEndpoints test
free: false
apiKey: {{ keyByAlias .credentials "testKey" }}

project:
  name: {{ .instance_name }}

cluster:
  name: {{ .instance_name }}
  providerBackupEnabled: {{ default "true" .backups }}
  providerSettings:
    providerName: {{ default "AZURE" .provider }}
    instanceSizeName: {{ default "M10" .instance_size }}
    regionName: {{ default "US_WEST_2" .region }}
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

privateEndpoints:
- provider: "AZURE"
  subscriptionID: fd01adff-b37e-4693-8497-83ecf183a145
  region: "EUROPE_NORTH"
  location: "northeurope"
  resourceGroup: svet-test
  virtualNetworkName: svet-test-vpc
  subnetName: default
  endpointName: svet-test-endpoint
