name: minimal-plan
description: This is a minimal plan for a cluster
free: true
apiKey: {{ json (index .Credentials.Projects .Project.ID) }}
project:
  id: {{ .Project.ID }}
cluster:
  name: {{ .Cluster.Name }}
  providerSettings:
    providerName: AWS
    instanceSizeName: M20
    regionName: US_EAST_2