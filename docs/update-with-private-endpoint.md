# Update existing cluster and add Private Endpoint

For this example two different plans will be used: [basic-plan](../samples/plans/sample_basic.yml.tpl) and [basic-plan-pe](../samples/plans/sample_basic_pe.yml.tpl). First we will create a service with no Private Endpoint and then we are going to `update` and add a Private Endpoint using the new plan.

1. Create a service with no Private Endpoint using the [basic-plan](../samples/plans/sample_basic.yml.tpl).

    > push

    ```bash
    cf push atlas-osb --no-start
    cf cups atlas-osb-keys -p ./samples/apikeys-config.json
    cf bind-service atlas-osb atlas-osb-keys
    cf set-env atlas-osb ATLAS_BROKER_TEMPLATEDIR ./samples/plans
    cf set-env atlas-osb BROKER_HOST 0.0.0.0
    cf set-env atlas-osb BROKER_PORT 8080
    cf set-env atlas-osb AZURE_CLIENT_ID ${AZURE_CLIENT_ID}
    cf set-env atlas-osb AZURE_CLIENT_SECRET ${AZURE_CLIENT_SECRET}
    cf set-env atlas-osb AZURE_TENANT_ID ${AZURE_TENANT_ID}
    cf start atlas-osb
    ```

    > create a service

    ```bash
    cf create-service-broker atlas-osb admin admin https://routes.url.com # make sure to fix the URL
    cf enable-service-access atlas
    cf create-service atlas basic-plan hello-atlas-osb
    ```

2. Update the service using the [basic-plan-pe](../samples/plans/sample_basic_pe.yml.tpl) with a Private Endpoint.

    > push

    Not required for this example, can skip

    > update the service

    ```bash
    cf update-service-broker atlas-osb admin admin https://routes.url.com # make sure to fix the URL
    cf update-service hello-atlas-osb -p basic-plan-pe
    ```

3. Wait for PE to be created on the existing cluster.
