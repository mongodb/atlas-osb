# Development

The broker is entirely written in Go and consists of a single executable, `main.go`, which makes use of two packages, `pkg/broker` and `pkg/atlas`. The executable runs an HTTP server which conforms to the [Open Service Broker API spec](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md).

The server is managed by a third-party library called [`brokerapi`](https://github.com/pivotal-cf/brokerapi). This library exposes a `ServerBroker` interface which we implement with `Broker` in `pkg/broker`. `pkg/atlas` contains a client for the Atlas API and `Broker` uses that client to translate incoming service broker requests to Atlas API calls.

**Do not clone this project to your $GOPATH.** This project uses Go modules which will be disabled if the project is built from the `$GOPATH`. If the project is built inside the `$GOPATH` then Go will fetch the dependencies from there as well. This could lead to incorrect versions and unreliable builds. When placed outside the `$GOPATH` dependencies will automatically be installed when the project is built.

## Testing

Please refer to the how-to documentation [action](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md)

## Releasing

The release process consists of publishing a new Github release with attached binaries as well as publishing a Docker image to [quay.io](https://quay.io). Evergreen can automatically build and publish the artifacts based on a tagged commit.

1. Go to the GitHub actions [page](https://github.com/mongodb/atlas-osb/actions?query=workflow%3A%22Create+GitHub+Release+Package+Manually%22)
2. Open "Create GitHub Release Package Manually" workflow.
3. Press "Run workflow" and choose parameters.
For example, our version is `v0.5.0-beta` and we want to update it to `v0.6.1-beta`. In that case we should write "-mp" in the "Version key" input field and "beta" in the "Add Postfix".

## Adding third-party dependencies

Please include their license in the notices/ directory.

## Setting up TLS in Kubernetes

To enable TLS, perform these steps before continuing with "Testing in Kubernetes".

1. Generate a self-signed certificate and private key by running `openssl req -newkey rsa:2048 -nodes -keyout key-x509 -days 365 -out cert`.
   When prompted for "Common Name", enter `atlas-service-broker.atlas`. All other fields can be left empty.
2. Create a new secret containing the key and cert by running `kubectl create secret generic aosb-tls --from-file=./key --from-file=./cert -n atlas`.
3. Update `samples/kubernetes/deployment.yaml` to mount the secret inside your pod in accordance with this guide: https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod.
   Also add the `BROKER_TLS_KEY_FILE` and `BROKER_TLS_CERT_FILE` environment variables to point to the mounted secret. If mounted to `/etc/tls_secret`
   the environment variables would be `/etc/tls_secret/key` and `/etc/tls_secret/cert`. Also change the service port from `80` to `443`.
4. Update `samples/kubernetes/service-broker.yaml` and add a `caBundle` field containing the base64 encoded contents of `cert`.
   Run `base64 < cert` to get the base64 string. Also update the `url` field to use `https`.

## Testing in Kubernetes

There are several ways to run Atlas-OSB in Kubernetes:

- with [GitHub Actions](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md#run-atlas-osb-with-github-action)
- with [Helm](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md#run-atlas-osb-with-helm)
- with [kubectl](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md#run-atlas-osb-with-kubectl)

### Run Atlas-OSB with GitHub Action

1. Install `act` tool
2. Create `.actrc` file as described in [HOWTO](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md)
3. Install the service catalog extension in Kubernetes, if it is not there yet:

```bash
act -j k8s-deploy-catalog
```

3. Run the following commands:

```bash
act -j k8s-demo-broker
act -j k8s-demo-instance
act -j k8s-demo-test
```

For more information about GitHub Actions, please follow [HOWTO](https://github.com/mongodb/atlas-osb/blob/master/.github/HOWTO.md)

### Run Atlas-OSB with Helm

Helm charts are located in `samples/helm/` folder

1. Helm and kubernetes must be pre-installed. [Installation sample](https://github.com/mongodb/atlas-osb/blob/master/.github/base-dockerfile/helpers/install_k8s_helm.sh)
2. Make sure Service Catalog is also pre-installed: run `dev/scripts/install-service-catalog.sh`
3. Deploy broker to k8s

   ```bash
   helm install "${K_BROKER}" \
      --set namespace="${K_NAMESPACE}" \
      --set image="quay.io/mongodb/atlas-osb:latest" \
      --set atlas.orgId="${ATLAS_ORG_ID}" \
      --set atlas.publicKey="${ATLAS_PUBLIC_KEY}" \
      --set atlas.privateKey="${ATLAS_PRIVATE_KEY}" \
      --set broker.auth.username="${K_DEFAULT_USER}" \
      --set broker.auth.password="${K_DEFAULT_PASS}" \
      samples/helm/broker/ --namespace "${K_NAMESPACE}" --wait --timeout 10m --create-namespace
   ```

4. Create a new service instance

   ```bash
   helm install "${K_SERVICE}" samples/helm/sample-service/ \
      --set broker.auth.username="${K_DEFAULT_USER}" \
      --set broker.auth.password="${K_DEFAULT_PASS}" \
      --namespace "${K_NAMESPACE}" --wait --timeout 60m
   ```

5. If necessary, install the test application. Set `service.name` as in the previous step for correct binding.

   ```bash
   helm install "${K_TEST_APP}" samples/helm/test-app/ \
      --set service.name="${K_SERVICE}" \
      --namespace "${K_NAMESPACE}" --wait --timeout 10m
   ```

Usage samples in .github/workflows/k8s-demo-*.yml

### Run Atlas-OSB with kubectl

Follow these steps to test the broker in a Kubernetes cluster. For local testing we recommend using [minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/). We also recommend using the [service catalog CLI](https://github.com/kubernetes-sigs/service-catalog/blob/master/docs/cli.md) (`svcat`) to control the service catalog.

1. Run `dev/scripts/install-service-catalog.sh` to install the service catalog extension in Kubernetes.
   Make sure you have Helm installed and configured before running.
2. Make sure the Service Catalog extension is installed in Kubernetes. Installation instructions can
   be found in the [Kubernetes docs](https://kubernetes.io/docs/tasks/service-catalog/install-service-catalog-using-helm/).
3. Build the Dockerfile and make the resulting image available in your cluster. If you are using
   Minikube `dev/scripts/minikube-build.sh` can be used to build the image using Minikube's Docker
   daemon. Update the deployment resource in `samples/kubernetes/deployment.yaml` to have
   `imagePullPolicy: Never` and update the `ATLAS_BASE_URL` to whichever environment you're testing against.
4. Create a new namespace `atlas` by running `kubectl create namespace atlas`.
5. Change a secret called `atlas-auth` containing the following keys:
   - `orgId` should be the Atlas group ID
   - `publicKey` should be the public key
   - `privateKey` should be the Atlas private key.
   run `kubectl apply -f samples/kubernetes/atlas-service-broker-auth.yaml -n atlas`
6. Update plan inside config-map `samples/kubernetes/deployment.yaml`
   and apply it to kubernetes `kubectl apply -f samples/kubernetes/config-map-plan.yaml -n atlas`
6. Deploy the service broker by running `kubectl apply -f samples/kubernetes/deployment.yaml -n atlas`. This will create
   a new deployment and a service of the image from step 2.
7. Register the service broker with the service catalog by running `kubectl apply -f samples/kubernetes/service-broker.yaml -n atlas`.
8. Make sure the broker is ready by running `svcat get brokers`.
9. A new instance can be provisioned by running `kubectl create -f samples/kubernetes/instance.yaml -n atlas`.
   The instance will be given the name `atlas-cluster-instance` and its status can be checked using `svcat get instances -n atlas`.
10. Once the instance is up and running, a binding can be created to gain access. A binding named
   `atlas-cluster-binding` can be created by running `kubectl create -f
   samples/kubernetes/binding.yaml -n atlas`. The binding credentials will automatically be stored in a secret
   of the same name.
11. After use, all bindings can be removed by running `svcat unbind atlas-cluser-instance -n atlas` and the
   cluster can be deprovisioned using `svcat deprovision atlas-cluster-instance -n atlas`.
