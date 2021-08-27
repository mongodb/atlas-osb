package test

import (
	"os"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/atlaskey"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
)

type Test struct {
	APIKeys    atlaskey.KeyList
	OrgName    string
	BrokerURL  string
	AppURL     string
	SpaceName  string
	BrokerApp  string
	Broker     string
	ServiceIns string
	TestApp    string
	PlanName   string
}

func NewTest() Test {
	id := utils.GenID()
	test := Test{}

	// from params.sh
	test.OrgName = os.Getenv("ORG_NAME") // do not change org name from param.sh. clean-failed GitHub action couldn't clean up properly otherwise
	test.SpaceName = os.Getenv("SPACE_NAME") + id
	test.BrokerApp = os.Getenv("BROKER_APP") + id
	test.Broker = os.Getenv("BROKER") + id
	test.ServiceIns = os.Getenv("SERVICE_ATLAS") + id
	test.TestApp = os.Getenv("TEST_SIMPLE_APP") + id
	test.PlanName = os.Getenv("TEST_PLAN")
	test.APIKeys = atlaskey.NewAtlasKeys()
	return test
}
