# Build the project before pushing to Cloud Foundry

To save resources it is possible to build the project and push the binary file to Cloud Foundry.

## Example Steps

> it is assumed you've read the main [README](../README.md) before following these steps

1. Build the project into [build/](../build) directory and push using [binary buildpack](https://docs.cloudfoundry.org/buildpacks/binary/index.html)
```sh
env GOOS=linux GOARCH=386 go build -o build/atlas-osb-linux .
cf push atlas-osb -c './build/atlas-osb-linux' -b binary_buildpack --no-start
```
2. Create keys service and bind to `atlas-osb`
```sh
cf cups atlas-osb-keys -p ./samples/apikeys-config.json
cf bind-service atlas-osb atlas-osb-keys
```
3. Set the required envs
```sh
cf set-env atlas-osb ATLAS_BROKER_TEMPLATEDIR ./samples/plans
cf set-env atlas-osb BROKER_HOST 0.0.0.0
cf set-env atlas-osb BROKER_PORT 8080
# Set the required Azure credentials for Private Link
cf set-env atlas-osb AZURE_CLIENT_ID ${AZURE_CLIENT_ID}
cf set-env atlas-osb AZURE_CLIENT_SECRET ${AZURE_CLIENT_SECRET}
cf set-env atlas-osb AZURE_TENANT_ID ${AZURE_TENANT_ID}
```
4. Start the service and create the service broker (make sure to fix the url)
```sh
cf start atlas-osb
cf create-service-broker atlas-osb admin admin https://routes.url.com
```
5. If `marketplace` is empty enable atlas service access
```sh
cf enable-service-access atlas
```
6. Create `hello-atlas-osb` using `basic-plan` 
```sh
cf create-service atlas basic-plan hello-atlas-osb
cf service hello-atlas-osb
```

## Potential Errors

> does not have authorization to perform action

```
The client 'XXXX-XX-XXXX' with object id 'XXXX-XX-XXXX'' does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/read' over scope '/subscriptions/ssss-cc-pppp/resourceGroups/test-group/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/default' or the scope is invalid. If access was recently granted, please refresh your credentials.
```

If you see a similar error it means your Azure App doesn't have the required permissions to create a Private Endpoint and you need to give it permissions. You need to look for a `Subscriptions` page on [Azure Portal](https://portal.azure.com/#home). Find `Access control (IAM)` in you subscription and use `Add role assignment` to give permissions to you Azure App (`Contributor` works for sure).

> Private Endpoint created but fails afterwards

Make sure there is no existing Private Endpoint on Azure side before, cause it will create a conflict. You can generate random name suffix in the `plan` file.
