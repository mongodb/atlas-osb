module github.com/mongodb/atlas-osb

go 1.11

replace go.mongodb.org/atlas => github.com/vasilevp/go-client-mongodb-atlas v0.7.3-0.20210413111111-8bb5160e00a2

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v53.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.17
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/TheZeroSlave/zapsentry v1.6.0
	github.com/alexflint/go-arg v1.3.1-0.20200806235247-96c756c382ed
	github.com/davecgh/go-spew v1.1.1
	github.com/drewolson/testflight v1.0.0 // indirect
	github.com/goccy/go-yaml v1.8.9
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-querystring v1.1.0
	github.com/gorilla/mux v1.8.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pivotal-cf/brokerapi v5.1.0+incompatible
	github.com/pkg/errors v0.9.1
	go.mongodb.org/atlas v0.11.0
	go.mongodb.org/mongo-driver v1.5.1
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/tools v0.0.0-20200103221440-774c71fcf114 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.4.0
	sigs.k8s.io/go-open-service-broker-client/v2 v2.0.0-20200706192557-3a0d26033ee6
)
