package test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/mongodb/atlas-osb/test/cfe2e/config"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/atlaskey"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/cf"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
	. "github.com/onsi/ginkgo"         // nolint
	. "github.com/onsi/gomega"         // nolint
	. "github.com/onsi/gomega/gbytes"  // nolint
	. "github.com/onsi/gomega/gexec"   // nolint
	. "github.com/onsi/gomega/gstruct" // nolint
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

func (t *Test) SaveLogs() {
	s := cfc.Cf("logs", t.BrokerApp, "--recent")
	Expect(s).Should(Exit(0))
	utils.SaveToFile(fmt.Sprintf("output/%s", t.BrokerApp), s.Out.Contents())
}

func (t *Test) Login() {
	By("Can login to CF and create organization")
	PCFKeys, err := cf.NewCF()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(PCFKeys).To(MatchFields(IgnoreExtras, Fields{
		"URL":      Not(BeEmpty()),
		"User":     Not(BeEmpty()),
		"Password": Not(BeEmpty()),
	}))
	Eventually(cfc.Cf("login", "-a", PCFKeys.URL, "-u", PCFKeys.User, "-p", PCFKeys.Password, "--skip-ssl-validation")).Should(Say("OK"))
	Eventually(cfc.Cf("create-org", t.OrgName)).Should(Say("OK"))
	Eventually(cfc.Cf("target", "-o", t.OrgName)).Should(Exit(0))
	Eventually(cfc.Cf("create-space", t.SpaceName)).Should(Exit(0))
	Eventually(cfc.Cf("target", "-s", t.SpaceName)).Should(Exit(0))
}

func (t *Test) PushBroker() {
	s := cfc.Cf("push", t.BrokerApp, "-p", "../../.", "--no-start") // ginkgo starts from test-root folder
	Eventually(s, config.CFEventuallyTimeoutMiddle, config.IntervalMiddle).Should(Exit(0))
	Eventually(s).Should(Say("down"))
}

func (t *Test) PushTestAppAndBindService() {
	testAppRepo := "https://github.com/leo-ri/simple-ruby.git"
	_, err := git.PlainClone("simple-ruby", false, &git.CloneOptions{
		URL:               testAppRepo,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get test application %s", t.AppURL)))
	}
	Eventually(cfc.Cf("push", t.TestApp, "-p", "./simple-ruby", "--no-start")).Should(Say("down"))
	Eventually(cfc.Cf("bind-service", t.TestApp, t.ServiceIns)).Should(Say("OK"))

	s := cfc.Cf("restart", t.TestApp)
	Eventually(s, config.CFEventuallyTimeoutMiddle, config.IntervalMiddle).Should(Exit(0))
	Eventually(s, config.CFEventuallyTimeoutMiddle, config.IntervalMiddle).Should(Say("running"))
	Eventually(cfc.Cf("app", t.TestApp), "5m", config.IntervalMiddle).Should(Say("running"))
	t.AppURL = string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
	Expect(t.AppURL).ShouldNot(BeEmpty())
}

func (t *Test) SetDefaultEnv() {
	Eventually(cfc.Cf("set-env", t.BrokerApp, "BROKER_LOG_LEVEL", "DEBUG")).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "BROKER_HOST", "0.0.0.0")).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "BROKER_PORT", "8080")).Should(Exit(0))
	cKey, _ := json.Marshal(t.APIKeys)
	Eventually(cfc.Cf("set-env", t.BrokerApp, "BROKER_APIKEYS", string(cKey))).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "ATLAS_BROKER_TEMPLATEDIR", config.TestPath)).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "BROKER_OSB_SERVICE_NAME", config.MarketPlaceName)).Should(Exit(0))
	// cloud-qa
	Eventually(cfc.Cf("set-env", t.BrokerApp, "ATLAS_BASE_URL", config.CloudQAHost)).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "REALM_BASE_URL", config.CloudQARealm)).Should(Exit(0))
}

func (t *Test) SetAzureEnv() {
	Eventually(cfc.Cf("set-env", t.BrokerApp, "AZURE_CLIENT_ID", os.Getenv("AZURE_CLIENT_ID"))).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "AZURE_CLIENT_SECRET", os.Getenv("AZURE_CLIENT_SECRET"))).Should(Exit(0))
	Eventually(cfc.Cf("set-env", t.BrokerApp, "AZURE_TENANT_ID", os.Getenv("AZURE_TENANT_ID"))).Should(Exit(0))
}

func (t *Test) RestartBrokerApp() {
	s := cfc.Cf("restart", t.BrokerApp)
	Eventually(s, config.CFEventuallyTimeoutMiddle, config.IntervalMiddle).Should(Say("running"))
	t.BrokerURL = "http://" + string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
}

func (t *Test) CreateServiceBroker() {
	By("Possible to create service-broker")
	GinkgoWriter.Write([]byte(t.BrokerURL))
	Eventually(cfc.Cf("create-service-broker", t.Broker, t.APIKeys.Broker.Username, t.APIKeys.Broker.Password,
		t.BrokerURL, "--space-scoped")).Should(Exit(0))
	Eventually(cfc.Cf("marketplace")).Should(Say(config.MarketPlaceName))
}

func (t *Test) CreateService() {
	orgID := t.APIKeys.Keys["TKey"]["OrgID"]
	c := fmt.Sprintf("{\"org_id\":\"%s\"}", orgID)
	s := cfc.Cf("create-service", config.MarketPlaceName, t.PlanName, t.ServiceIns, "-c", c)
	Eventually(s).Should(Exit(0))
	Eventually(s).Should(Say("OK"))
	Eventually(s).ShouldNot(Say("Service instance already exists"))
	t.WaitServiceStatus("create succeeded")
}

// CreateServiceKey - create-service-key command with -c key
// config samples:
// '{"user" : {"roles" : [ { "roleName" : "atlasAdmin", "databaseName" : "admin" } ] } }'
// '{"user" : {"roles" : [ { "roleName" : "read", "databaseName" : "admin"} ] } }'
func (t *Test) CreateServiceKey(config, keyName string) {
	if config == "" {
		config = "{}"
	}
	Eventually(cfc.Cf("create-service-key", t.ServiceIns, keyName, "-c", config)).Should(Say("OK"))
}

func (t *Test) DeleteServiceKey(keyName string) {
	s := cfc.Cf("delete-service-key", t.ServiceIns, keyName, "-f")
	Eventually(s).Should(Exit(0))
}

func (t *Test) DeleteServiceKeys() {
	s := cfc.Cf("service-keys", t.ServiceIns)
	Eventually(s).Should(Exit(0))
	re := regexp.MustCompile("name\n(.+){0,1}\n{0,1}(.+){0,1}\n{0,1}(.+){0,1}\n{0,1}(.+){0,1}")
	serviceKeys := re.FindStringSubmatch(string(s.Out.Contents()))
	if len(serviceKeys) > 1 {
		for _, key := range serviceKeys[1:] {
			if (len(key) > 0) && (key != "name") {
				t.DeleteServiceKey(key)
			}
		}
	}
}

func (t *Test) UpgradeClusterConfig() {
	Eventually(cfc.Cf("update-service", t.ServiceIns, "-c", t.UpdateType)).Should(Say("OK"))
}
