module github.com/mongodb/mongodb-atlas-service-broker

go 1.11

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/davecgh/go-spew v1.1.1
	github.com/drewolson/testflight v1.0.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-test/deep v1.0.1
	github.com/goccy/go-yaml v1.7.18
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/kubernetes-incubator/service-catalog v0.2.1
	github.com/kubernetes-sigs/service-catalog v0.2.1
	github.com/mitchellh/mapstructure v1.3.3
	github.com/mongodb/go-client-mongodb-atlas v0.3.0
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pivotal-cf/brokerapi v5.1.0+incompatible
	github.com/pkg/errors v0.8.1 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.0.4
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190806064354-8b51d7113622
	k8s.io/apimachinery v0.0.0-20190802060556-6fa4771c83b3
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1 // indirect
	sigs.k8s.io/go-open-service-broker-client/v2 v2.0.0-20200706192557-3a0d26033ee6
)

replace github.com/mongodb/go-client => ../go-client-mongodb-atlas
