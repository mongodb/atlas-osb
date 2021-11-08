module github.com/mongodb/atlas-osb

go 1.11

replace go.mongodb.org/atlas => github.com/vasilevp/go-client-mongodb-atlas v0.7.3-0.20210413111111-8bb5160e00a2

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v53.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.22
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/TheZeroSlave/zapsentry v1.6.0
	github.com/alexflint/go-arg v1.3.1-0.20200806235247-96c756c382ed
	github.com/davecgh/go-spew v1.1.1
	github.com/drewolson/testflight v1.0.0 // indirect
	github.com/go-git/go-git/v5 v5.4.2
	github.com/goccy/go-yaml v1.8.9
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/go-querystring v1.1.0
	github.com/gorilla/mux v1.8.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/mongodb-forks/digest v1.0.2
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pivotal-cf-experimental/cf-test-helpers v0.0.0-20170428144005-e56b6ec41da9
	github.com/pivotal-cf/brokerapi v5.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.2.0
	go.mongodb.org/atlas v0.7.2
	go.mongodb.org/mongo-driver v1.5.1
	go.uber.org/zap v1.16.0
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/tools v0.0.0-20210101214203-2dba1e4ea05c // indirect
	gopkg.in/yaml.v2 v2.4.0
	sigs.k8s.io/go-open-service-broker-client/v2 v2.0.0-20200706192557-3a0d26033ee6
)
