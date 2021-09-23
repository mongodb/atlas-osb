package cfe2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"
	"go.mongodb.org/atlas/mongodbatlas"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/cf"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
	cfc "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

type azure struct {
	clientID string
	clientSecret string
	tenantID string
}

var _ = Describe("Feature: Atlas broker supports basic template[pe-flow]", func() {
	var testFlow test.Test
	var PCFKeys cf.CF
	var az azure

	_ = BeforeEach(func() {
		By("Check enviroment", func() {
			Expect(os.Getenv("AZURE_CLIENT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_ID env")
			Expect(os.Getenv("AZURE_CLIENT_SECRET")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_SECRET env")
			Expect(os.Getenv("AZURE_TENANT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_TENANT_ID env")
		})
		By("Set up", func() {
			testFlow = test.NewTest()
			Expect(testFlow.APIKeys.Keys[TKey]).Should(HaveKeyWithValue("publicKey", Not(BeEmpty())))
			PCFKeys = cf.NewCF()
			Expect(PCFKeys).To(MatchFields(IgnoreExtras, Fields{
				"URL":      Not(BeEmpty()),
				"User":     Not(BeEmpty()),
				"Password": Not(BeEmpty()),
			}))
			az.clientID = os.Getenv("AZURE_CLIENT_ID")
			az.clientSecret = os.Getenv("AZURE_CLIENT_SECRET")
			az.tenantID = os.Getenv("AZURE_TENANT_ID")
		})
	})

	_ = AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			s := cfc.Cf("logs", testFlow.BrokerApp, "--recent")
			Expect(s).Should(Exit(0))
			utils.SaveToFile(fmt.Sprintf("output/%s", testFlow.BrokerApp), s.Out.Contents())
			testFlow.DeleteResources()
		}
	})

	When("Given names and plan template AZURE", func() {
		It("Should pass flow", func() {
			By("Can login to CF and create organization", func() {
				Eventually(cfc.Cf("login", "-a", PCFKeys.URL, "-u", PCFKeys.User, "-p", PCFKeys.Password, "--skip-ssl-validation")).Should(Say("OK"))
				Eventually(cfc.Cf("create-org", testFlow.OrgName)).Should(Say("OK"))
				Eventually(cfc.Cf("target", "-o", testFlow.OrgName)).Should(Exit(0))
				Eventually(cfc.Cf("create-space", testFlow.SpaceName)).Should(Exit(0))
				Eventually(cfc.Cf("target", "-s", testFlow.SpaceName)).Should(Exit(0))
			})
			By("Can create service broker from repo and setup env", func() {
				s := cfc.Cf("push", testFlow.BrokerApp, "-p", "../../.", "--no-start") // ginkgo starts from test-root folder
				Eventually(s, CFEventuallyTimeoutMiddle, IntervalMiddle).Should(Exit(0))
				Eventually(s).Should(Say("down"))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "BROKER_LOG_LEVEL", "DEBUG")).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "BROKER_HOST", "0.0.0.0")).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "BROKER_PORT", "8080")).Should(Exit(0))
				cKey, _ := json.Marshal(testFlow.APIKeys)
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "BROKER_APIKEYS", string(cKey))).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "ATLAS_BROKER_TEMPLATEDIR", tPath)).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "BROKER_OSB_SERVICE_NAME", mPlaceName)).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "AZURE_CLIENT_ID", az.clientID)).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "AZURE_CLIENT_SECRET", az.clientSecret)).Should(Exit(0))
				Eventually(cfc.Cf("set-env", testFlow.BrokerApp, "AZURE_TENANT_ID", az.tenantID)).Should(Exit(0))

				s = cfc.Cf("restart", testFlow.BrokerApp)
				Eventually(s, CFEventuallyTimeoutMiddle, IntervalMiddle).Should(Say("running"))
				testFlow.BrokerURL = "http://" + string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
			})
			By("Possible to create service-broker", func() {
				GinkgoWriter.Write([]byte(testFlow.BrokerURL))
				Eventually(cfc.Cf("create-service-broker", testFlow.Broker, testFlow.APIKeys.Broker.Username, testFlow.APIKeys.Broker.Password,
					testFlow.BrokerURL, "--space-scoped")).Should(Exit(0))
				Eventually(cfc.Cf("marketplace")).Should(Say(mPlaceName))
			})

			By("Possible to create a service", func() {
				orgID := testFlow.APIKeys.Keys["TKey"]["OrgID"]
				c := fmt.Sprintf("{\"org_id\":\"%s\"}", orgID)
				s := cfc.Cf("create-service", mPlaceName, testFlow.PlanName, testFlow.ServiceIns, "-c", c)
				Eventually(s).Should(Exit(0))
				Eventually(s).Should(Say("OK"))
				Eventually(s).ShouldNot(Say("Service instance already exists"))
				testFlow.WaitServiceStatus("create succeeded")
			})

			By("Possible to create service-key", func() {
				Eventually(cfc.Cf("create-service-key", testFlow.ServiceIns, "atlasKey")).Should(Say("OK"))
				// '{"user" : { "roles" : [ { "roleName":"atlasAdmin", "databaseName" : "admin" } ] } }'
				GinkgoWriter.Write([]byte("Possible to create service-key. Check is not ready")) // TODO !
			})

			By("Check PE status", func() {
				AC := AClient(testFlow.APIKeys.Keys[TKey])
				projectInfo, _, err := AC.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
				// Expected
				// |         <string>: inst-a0b929f8c848e5iidhb9o1l5
				// |     to equal
				// |         <string>: inst-8c848e5iidhb9o1l5
				// projectInfo, _, err := AC.Projects.GetOneProjectByName(context.Background(), "inst-8c848e5iidhb9o1l5")
				Expect(err).ShouldNot(HaveOccurred())

				pConnection, _, err := AC.PrivateEndpoints.List(context.Background(), projectInfo.ID, "AZURE", &mongodbatlas.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				GinkgoWriter.Write([]byte(pConnection[0].Status))
				Expect(pConnection[0].Status).Should(Equal("AVAILABLE"))


				// ctx context.Context, groupID, cloudProvider string, listOptions *ListOptions
				// privateID := ""
				// AC.PrivateEndpoints.Get(context.Background(), testData.APIKeys.Keys[TKey]["orgID"], "AZURE", privateID)
			})




			// By("Can scale cluster size", func() {
			// 	newSize := "M20"
			// 	Eventually(cfc.Cf("update-service", testData.ServiceIns, "-c", "{\"instance_size\":\"M20\"}")).Should(Say("OK"))
			// 	waitServiceStatus(testData.ServiceIns, "update succeeded")

			// 	// get the real size
			// 	AC := AClient(testData.APIKeys.Keys[TKey])
			// 	projectInfo, _, err := AC.Projects.GetOneProjectByName(context.Background(), testData.ServiceIns)
			// 	Expect(err).ShouldNot(HaveOccurred())
			// 	Expect(projectInfo.ID).ShouldNot(BeEmpty())
			// 	clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, testData.ServiceIns)
			// 	Expect(clusterInfo.ProviderSettings.InstanceSizeName).Should(Equal(newSize))
			// })

			testFlow.DeleteResources()
		})
	})
})
