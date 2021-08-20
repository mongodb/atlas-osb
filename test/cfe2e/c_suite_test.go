package cfe2e

import (
	"os"
	"testing"
	"time"

	"github.com/mongodb-forks/digest"
	c "github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"go.mongodb.org/atlas/mongodbatlas"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

/*
Following environment varialble should be present:
INPUT_ATLAS_ORG_ID
INPUT_ATLAS_PRIVATE_KEY
INPUT_ATLAS_PUBLIC_KEY
INPUT_CF_API
INPUT_CF_USER
INPUT_CF_PASSWORD
These variables are copies of github secrets

These set up by pipeline (param.sh)
ORG_NAME
SPACE_NAME
BROKER
BROKER_APP
TEST_SIMPLE_APP
SERVICE_ATLAS
*/

const (
	CFEventuallyTimeout   = 60 * time.Second
	CFConsistentlyTimeout = 60 * time.Millisecond
	TKey                  = "testKey" // TODO get it from the plan
	tPath                 = "./test/cfe2e/data"
	mPlaceName            = "atlas"
)

var (
	homeDir    string //nolint
	orgName    string
	brokerURL  string
	appURL     string
	spaceName  string
	brokerApp  string
	broker     string
	serviceIns string
	testApp    string

	planName = "override-bind-db-plan"
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Atlas Broker")
}

var _ = BeforeSuite(func() {
	GinkgoWriter.Write([]byte("==============================Before==============================\n"))
	SetDefaultEventuallyTimeout(CFEventuallyTimeout)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeout)

	checkupCFinputs()
	setUp()
	GinkgoWriter.Write([]byte("========================End of Before==============================\n"))
})

func checkupCFinputs() {
	Expect(os.Getenv("INPUT_CF_URL")).ToNot(BeEmpty(), "Please, set up INPUT_CF_URL env")
	Expect(os.Getenv("INPUT_CF_USER")).ToNot(BeEmpty(), "Please, set up INPUT_CF_USER env")
	Expect(os.Getenv("INPUT_CF_PASSWORD")).ToNot(BeEmpty(), "Please, set up INPUT_CF_PASSWORD env")
}

func setUp() {
	orgName = os.Getenv("ORG_NAME")
	spaceName = os.Getenv("SPACE_NAME")
	brokerApp = os.Getenv("BROKER_APP")
	broker = os.Getenv("BROKER")
	serviceIns = os.Getenv("SERVICE_ATLAS")
	testApp = os.Getenv("TEST_SIMPLE_APP")
	Expect(orgName).ToNot(BeEmpty())
	Expect(spaceName).ToNot(BeEmpty())
	Expect(brokerApp).ToNot(BeEmpty())
	Expect(serviceIns).ToNot(BeEmpty())
	Expect(testApp).ToNot(BeEmpty())
}

func AClient(keys c.Credential) *mongodbatlas.Client {
	t := digest.NewTransport(keys["publicKey"], keys["privateKey"])
	tc, err := t.Client()
	if err != nil {
		panic(err)
	}
	return mongodbatlas.NewClient(tc)
}
