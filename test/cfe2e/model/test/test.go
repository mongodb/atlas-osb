package test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/atlaskey"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	cfc "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
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
	UpdateType string
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
	test.UpdateType = os.Getenv("TEST_UPDATE_TYPE")
	return test
}

func WaitForDelete() {
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
func (t *Test) WaitServiceStatus(expectedStatus string) {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) // TODO
		try++
		s := cfc.Cf("service", t.ServiceIns)
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

func (t *Test) DeleteResources() {
	By("Possible to delete service-key", func() {
		Eventually(cfc.Cf("delete-service-key", t.ServiceIns, "atlasKey", "-f")).Should(Say("OK"))
	})
	// By("Possible to unbind service", func() {
	// 	Eventually(cfc.Cf("unbind-service", t.TestApp, t.ServiceIns)).Should(Say("OK"))
	// })
	// By("Possible to delete test application after use", func() {
	// 	Eventually(cfc.Cf("delete", t.TestApp, "-f")).Should(Say("OK"))
	// })
	By("Possible to delete service", func() {
		Eventually(cfc.Cf("delete-service", t.ServiceIns, "-f")).Should(Say("OK"))
		WaitForDelete()
	})
	By("Possible to delete Service broker", func() {
		Eventually(cfc.Cf("delete-service-broker", t.Broker, "-f")).Should(Say("OK"))
	})
	By("Possible to delete broker application", func() {
		Eventually(cfc.Cf("delete", t.BrokerApp, "-f")).Should(Say("OK"))
	})
}

func (t *Test) DeleteApplicationResources() {
	By("Possible to unbind service", func() {
		Eventually(cfc.Cf("unbind-service", t.TestApp, t.ServiceIns)).Should(Say("OK"))
	})
	By("Possible to delete test application after use", func() {
		Eventually(cfc.Cf("delete", t.TestApp, "-f")).Should(Say("OK"))
	})
}
