# <img src="https://webassets.mongodb.com/_com_assets/cms/vectors-anchor-circle-mydmar539a.svg"/> MongoDB Atlas-OSB 


This project is a new version of the original [MongoDB Atlas Serivce Broker](https://github.com/mongodb/mongodb-atlas-service-broker).

The new atlas-osb adds the ability to define your own plans from Atlas resource templates. This new and powerful feature brings the broker to a new level of production-readiness. One simple `create-service` command is all you need to provision a complete Atlas deployment including a Project, Cluster, Database user, firewall access, and more.

The atlas-osb also adds a new level of security and configurability with the ability to deploy the broker with multiple Atlas apikeys and also integrate these keys into CredHub and Secrets.

Use the Atlas Service Broker to connect to [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) from any platform which supports the [Open Service Broker API](https://www.openservicebrokerapi.org/), such as [Kubernetes](https://kubernetes.io/) and [Pivotal Cloud Foundry](https://pivotal.io/open-service-broker).

- Provision managed MongoDB clusters on Atlas directly from your platform of choice. Includes support for all cluster configuration settings and cloud providers available on Atlas.
- Manage and scale clusters without leaving your platform.
- Create bindings to allow your applications access to clusters.

## Documentation

Our docs are in flight. See [docs](/docs).

Best place to start for cf: [getting-started-cf-atlas-osb.md](/docs/getting-started-cf-atlas-osb.md)

New plan template spec: [custom-plans.md](/docs/custom-plans.md)


## Configuration

Configuration is handled with environment variables. Logs are written to
`stdout/stderr` as appropriate and each line is in a structured JSON format.

| Variable | Default | Description |
| -------- | ------- | ----------- |
| ATLAS_BASE_URL | `https://cloud.mongodb.com` | Base URL used for Atlas API connections |
| BROKER_HOST | `127.0.0.1` | Address which the broker server listens on |
| BROKER_PORT | `4000` | Port which the broker server listens on |
| BROKER_LOG_LEVEL | `INFO` | Accepted values: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| BROKER_TLS_CERT_FILE | | Path to a certificate file to use for TLS. Leave empty to disable TLS. |
| BROKER_TLS_KEY_FILE | | Path to private key file to use for TLS. Leave empty to disable TLS. |
| PROVIDERS_WHITELIST_FILE | | Path to a JSON file containing limitations for providers and their plans. |
| BROKER_APIKEYS | | Path to file or JSON string containing credentials.
| ATLAS_BROKER_TEMPLATEDIR | | Path to folder containing plans e.g. ./samples/plans |

## License

See [LICENSE](LICENSE). Licenses for all third-party dependencies are included in [notices](notices).

## Development

Information regarding development, testing, and releasing can be found in the [development documentation](dev).
