package cfe2e

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/mongodb-forks/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	c "github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

/*
Following environment varialble should be present:
INPUT_ATLAS_PROJECT_ID
INPUT_ATLAS_ORG_ID
INPUT_ATLAS_PRIVATE_KEY
INPUT_ATLAS_PUBLIC_KEY
INPUT_PCF_URL
INPUT_PCF_USER
INPUT_PCF_PASSWORD
INPUT_CF_API
INPUT_CF_USER
INPUT_CF_PASSWORD
These variables are copies of github secrets
*/

type KeyList struct {
	Keys   map[string]c.APIKey `json:"keys"`
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
)

var (
	homeDir string //nolint
	APIKeys KeyList
	PCFKeys PCF
	AC      *mongodbatlas.Client
	Commit  string
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Atlas Broker")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	GinkgoWriter.Write([]byte("==============================Global FIRST Node Synchronized Before Each==============================\n")) //nolint
	GinkgoWriter.Write([]byte("SetUp Global Timeout\n"))//nolint
	SetDefaultEventuallyTimeout(CFEventuallyTimeout)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeout)
	GinkgoWriter.Write([]byte("==============================End of Global FIRST Node Synchronized Before Each=======================\n")) //nolint
	return nil
}, func(_ []byte) {
	GinkgoWriter.Write([]byte(fmt.Sprintf("==============================Global Node %d Synchronized Before Each==============================", GinkgoParallelNode())))//nolint
	if GinkgoParallelNode() != 1 {
		Fail("Please Test suite cannot run in parallel")
	}
	GinkgoWriter.Write([]byte(fmt.Sprintf("==============================End of Global Node %d Synchronized Before Each========================", GinkgoParallelNode())))//nolint
})

var _ = BeforeEach(func() {
	GinkgoWriter.Write([]byte("==============================Global Before Each==============================\n")) //nolint
	setUp()
	GinkgoWriter.Write([]byte("========================End of Global Before Each==============================\n")) //nolint
})

func setUp() {
	Commit = os.Getenv("GITHUB_REF") // refs/heads/sample
	Commit = string(regexp.MustCompile(".+/(.+)").FindSubmatch([]byte(Commit))[1])
	Expect(Commit).ToNot(BeEmpty())

	PCFKeys = PCF{
		Endpoint: os.Getenv("INPUT_CF_API"),      //TODO do we need opsman pass? INPUT_PCF_URL
		User:     os.Getenv("INPUT_CF_USER"),     //TODO do we need opsman pass? INPUT_PCF_USER
		Password: os.Getenv("INPUT_CF_PASSWORD"), //TODO do we need opsman pass? INPUT_PCCF_PASSWORD
	}

	keys := c.APIKey{
		OrgID:      os.Getenv("INPUT_ATLAS_ORG_ID"),
		PublicKey:  os.Getenv("INPUT_ATLAS_PUBLIC_KEY"),
		PrivateKey: os.Getenv("INPUT_ATLAS_PRIVATE_KEY"),
	}

	APIKeys = KeyList{
		Keys: map[string]c.APIKey{
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
	Expect(APIKeys.Keys[TKey]).To(MatchFields(IgnoreExtras, Fields{
		"OrgID":      Not(BeEmpty()),
		"PublicKey":  Not(BeEmpty()),
		"PrivateKey": Not(BeEmpty()),
	}))
	Expect(APIKeys.Broker).To(PointTo(MatchFields(IgnoreExtras, Fields{
		"Username": Not(BeEmpty()),
		"Password": Not(BeEmpty()),
	})))
}

func AClient() *mongodbatlas.Client {
	t := digest.NewTransport(APIKeys.Keys[TKey].PublicKey, APIKeys.Keys[TKey].PrivateKey)
	tc, err := t.Client()
	if err != nil {
		panic(err)
	}
	return mongodbatlas.NewClient(tc)
}
