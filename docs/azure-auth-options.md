# Azure Authorization Options

Credentials for Azure Authorization are taken from the environment variables.

Supported options are: client credentials or certificate, username and password, or MSI.
 
### Input Priority

During authorization different auth options are tried in this order:

1. Client credentials
2. Client certificate
3. Username password
4. MSI

### Required inputs for each option

Pick one authorization option and supply all the required variables for the chosen way.

#### Client credentials
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_TENANT_ID`

#### Client certificate
- `AZURE_CERTIFICATE_PATH`
- `AZURE_CERTIFICATE_PASSWORD`
- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`

#### Username \& password
- `AZURE_USERNAME`
- `AZURE_PASSWORD`
- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`

#### MSI
- `AZURE_AD_RESOURCE`
- `AZURE_CLIENT_ID`
