package cfe2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mongodb-forks/digest"
	c "github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	cfc "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"go.mongodb.org/atlas/mongodbatlas"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
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
	CFEventuallyTimeoutDefault   = 60 * time.Second
	CFConsistentlyTimeoutDefault = 60 * time.Millisecond
	CFEventuallyTimeoutMiddle    = 10 * time.Minute
	IntervalMiddle               = 10 * time.Second

	// cf timouts
	CFStagingTimeout  = 15
	CFStartingTimeout = 15

	TKey       = "testKey" // TODO get it from the plan
	tPath      = "./test/cfe2e/data"
	mPlaceName = "atlas"
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Atlas Broker")
}

var _ = BeforeSuite(func() {
	GinkgoWriter.Write([]byte("==============================Before==============================\n"))
	SetDefaultEventuallyTimeout(CFEventuallyTimeoutDefault)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeoutDefault)
	checkupCFinputs()
	setUp()
	GinkgoWriter.Write([]byte("========================End of Before==============================\n"))
})

func setUp() {
	os.Setenv("CF_STAGING_TIMEOUT", fmt.Sprint(CFStagingTimeout))
	os.Setenv("CF_STARTUP_TIMEOUT", fmt.Sprint(CFStartingTimeout))
}

func checkupCFinputs() {
	Expect(os.Getenv("INPUT_CF_URL")).ToNot(BeEmpty(), "Please, set up INPUT_CF_URL env")
	Expect(os.Getenv("INPUT_CF_USER")).ToNot(BeEmpty(), "Please, set up INPUT_CF_USER env")
	Expect(os.Getenv("INPUT_CF_PASSWORD")).ToNot(BeEmpty(), "Please, set up INPUT_CF_PASSWORD env")

	Expect(os.Getenv("ORG_NAME")).ToNot(BeEmpty(), "Please, use param.sh or set up ORG_NAME")
	Expect(os.Getenv("BROKER_APP")).ToNot(BeEmpty(), "Please, use param.sh or set up BROKER_APP")
	Expect(os.Getenv("BROKER")).ToNot(BeEmpty(), "Please, use param.sh or set up BROKER")
	Expect(os.Getenv("TEST_SIMPLE_APP")).ToNot(BeEmpty(), "Please, use param.sh or set up TEST_SIMPLE_APP")
	Expect(os.Getenv("TEST_PLAN")).ToNot(BeEmpty(), "Please, set up TEST_PLAN env, name of the plan in test/data folder")
}

func AClient(keys c.Credential) *mongodbatlas.Client {
	t := digest.NewTransport(keys["publicKey"], keys["privateKey"])
	tc, err := t.Client()
	if err != nil {
		panic(err)
	}
	return mongodbatlas.NewClient(tc)
}

// TODO move
func waitForDelete() {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) // TODO
		try++
		session := cfc.Cf("services")
		EventuallyWithOffset(1, session).Should(Exit(0))
		isDeleted := strings.Contains(string(session.Out.Contents()), "No services found")
		GinkgoWriter.Write([]byte(fmt.Sprintf("Waiting for deletion (try #%d)", try)))

		if isDeleted {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 13 { // TODO what is our req. for awaiting
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			ExpectWithOffset(1, true).Should(Equal(false)) // TODO call fail
		}
	}
}

// waitStatus wait until status is appear
func waitServiceStatus(serviceName string, expectedStatus string) {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) // TODO
		try++
		s := cfc.Cf("service", serviceName)
		EventuallyWithOffset(1, s).Should(Exit(0))
		status := string(regexp.MustCompile(`status:\s+(.+)\s+`).FindSubmatch(s.Out.Contents())[1])
		GinkgoWriter.Write([]byte(fmt.Sprintf("Status is %s (try #%d)", status, try)))
		if status == expectedStatus {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 30 { // TODO req?
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			ExpectWithOffset(1, true).Should(Equal(false)) // TODO call fail
		}
	}
}

func getBackupStateFromPlanConfig(path string) bool {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	match := "providerBackupEnabled: .+ \"(.+)\" .+"
	backup, _ := strconv.ParseBool(string(regexp.MustCompile(match).FindSubmatch(config)[1]))
	return backup
}

func deleteResources(testData test.Test) {
	By("Possible to delete service-key", func() {
		Eventually(cfc.Cf("delete-service-key", testData.ServiceIns, "atlasKey", "-f")).Should(Say("OK"))
	})
	By("Possible to unbind service", func() {
		Eventually(cfc.Cf("unbind-service", testData.TestApp, testData.ServiceIns)).Should(Say("OK"))
	})
	By("Possible to delete test application after use", func() {
		Eventually(cfc.Cf("delete", testData.TestApp, "-f")).Should(Say("OK"))
	})
	By("Possible to delete service", func() {
		Eventually(cfc.Cf("delete-service", testData.ServiceIns, "-f")).Should(Say("OK"))
		waitForDelete()
	})
	By("Possible to delete Service broker", func() {
		Eventually(cfc.Cf("delete-service-broker", testData.Broker, "-f")).Should(Say("OK"))
	})
	By("Possible to delete broker application", func() {
		Eventually(cfc.Cf("delete", testData.BrokerApp, "-f")).Should(Say("OK"))
	})
}
