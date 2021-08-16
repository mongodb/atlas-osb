package cfe2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mongodb-forks/digest"
	c "github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"go.mongodb.org/atlas/mongodbatlas"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

type KeyList struct {
	Keys   map[string]c.Credential `json:"keys"`
	Broker *c.BrokerAuth       `json:"broker"`
}
type PCF struct {
	Endpoint string
	User     string
	Password string
}

const (
	CFEventuallyTimeout   = 60 * time.Second
	CFConsistentlyTimeout = 60 * time.Millisecond
	TKey                  = "testKey" //TODO get it from the plan
	tPath                 = "./test/cfe2e/data"
	mPlaceName            = "atlas"
)

var (
	homeDir    string //nolint
	APIKeys    KeyList
	PCFKeys    PCF
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

var _ = SynchronizedBeforeSuite(func() []byte {
	GinkgoWriter.Write([]byte("==============================Global FIRST Node Synchronized Before Each==============================\n"))
	GinkgoWriter.Write([]byte("SetUp Global Timeout\n"))
	SetDefaultEventuallyTimeout(CFEventuallyTimeout)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeout)
	setUp()
	GinkgoWriter.Write([]byte("==============================End of Global FIRST Node Synchronized Before Each=======================\n"))
	return nil
}, func(_ []byte) {
	GinkgoWriter.Write([]byte(fmt.Sprintf("==============================Global Node %d Synchronized Before Each==============================\n", GinkgoParallelNode())))
	if GinkgoParallelNode() != 1 {
		Fail("Please Test suite cannot run in parallel")
	}
	GinkgoWriter.Write([]byte(fmt.Sprintf("==============================End of Global Node %d Synchronized Before Each========================\n", GinkgoParallelNode())))
})

var _ = BeforeEach(func() {
	GinkgoWriter.Write([]byte("==============================Global Before Each==============================\n"))
	// setUp()
	GinkgoWriter.Write([]byte("========================End of Global Before Each==============================\n"))
})

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

	PCFKeys = PCF{
		Endpoint: os.Getenv("INPUT_CF_API"),
		User:     os.Getenv("INPUT_CF_USER"),
		Password: os.Getenv("INPUT_CF_PASSWORD"),
	}

	keys := c.Credential{
		"OrgID":      os.Getenv("INPUT_ATLAS_ORG_ID"),
		"PublicKey":  os.Getenv("INPUT_ATLAS_PUBLIC_KEY"),
		"PrivateKey": os.Getenv("INPUT_ATLAS_PRIVATE_KEY"),
	}

	APIKeys = KeyList{
		Keys: map[string]c.Credential{
			TKey: keys,
		},
		Broker: &c.BrokerAuth{
			Username: "admin",
			Password: "admin",
		},
	}
	//TODO check fails
	Expect(PCFKeys).To(MatchFields(IgnoreExtras, Fields{
		"Endpoint": Not(BeEmpty()),
		"User":     Not(BeEmpty()),
		"Password": Not(BeEmpty()),
	}))

	Expect(APIKeys.Keys[TKey]).Should(HaveKeyWithValue("OrgID", Not(BeEmpty())))
	Expect(APIKeys.Keys[TKey]).Should(HaveKeyWithValue("PublicKey", Not(BeEmpty())))
	Expect(APIKeys.Keys[TKey]).Should(HaveKeyWithValue("PrivateKey", Not(BeEmpty())))

	Expect(APIKeys.Broker).To(PointTo(MatchFields(IgnoreExtras, Fields{
		"Username": Not(BeEmpty()),
		"Password": Not(BeEmpty()),
	})))
}

func AClient() *mongodbatlas.Client {
	t := digest.NewTransport(APIKeys.Keys["TKey"]["PublicKey"], APIKeys.Keys["TKey"]["PrivateKey"])
	tc, err := t.Client()
	if err != nil {
		panic(err)
	}
	return mongodbatlas.NewClient(tc)
}
