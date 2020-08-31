module github.com/mongodb/mongodb-atlas-service-broker

go 1.11

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/alexflint/go-arg v1.3.1-0.20200806235247-96c756c382ed
	github.com/davecgh/go-spew v1.1.1
	github.com/go-git/go-git/v5 v5.1.0
	github.com/goccy/go-yaml v1.8.0
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/kubernetes-incubator/service-catalog v0.2.1
	github.com/kubernetes-sigs/service-catalog v0.2.1
	github.com/mitchellh/mapstructure v1.3.3
	github.com/mongodb-forks/digest v1.0.1
	github.com/mongodb/atlas-osb v0.5.0 // indirect
	github.com/mongodb/go-client-mongodb-atlas v0.3.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pivotal-cf-experimental/cf-test-helpers v0.0.0-20170428144005-e56b6ec41da9
	github.com/pivotal-cf/brokerapi v5.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	go.mongodb.org/mongo-driver v1.4.0
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.0.0-20190806064354-8b51d7113622
	k8s.io/apimachinery v0.0.0-20190802060556-6fa4771c83b3
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	sigs.k8s.io/go-open-service-broker-client/v2 v2.0.0-20200706192557-3a0d26033ee6
)

replace github.com/mongodb/go-client => ../go-client-mongodb-atlas
