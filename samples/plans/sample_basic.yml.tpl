name: basic-plan
description: "This is the `Basic Plan` template for 1 project, 1 cluster, 1 dbuser, and 1 secure connection."
free: true
apiKey: {{ mustToJson (randelem .Credentials.Orgs) }}
project:
  name: {{ .instance_name }}
  desc: Created from a template
cluster:
  name: {{ .instance_name }}
  providerSettings:
    providerName: {{ default "AWS" .provider }}
    instanceSizeName: {{ default "M20" .instance_size }}
    regionName: {{ default "US_EAST_1" .region }}
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
