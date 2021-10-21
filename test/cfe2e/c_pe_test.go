package cfe2e

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mongodb/atlas-osb/test/cfe2e/config"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/atlasclient"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
)

var _ = Describe("Feature: Atlas broker supports basic template[pe-flow]", func() {
	var testFlow test.Test

	_ = BeforeEach(func() {
		By("Check environment", func() {
			Expect(os.Getenv("AZURE_CLIENT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_ID env")
			Expect(os.Getenv("AZURE_CLIENT_SECRET")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_SECRET env")
			Expect(os.Getenv("AZURE_TENANT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_TENANT_ID env")
		})
		By("Set up", func() {
			testFlow = test.NewTest()
			Expect(testFlow.APIKeys.Keys[config.TKey]).Should(HaveKeyWithValue("publicKey", Not(BeEmpty())))
		})
	})

	_ = AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			testFlow.SaveLogs()
			testFlow.DeleteResources()
		}
	})

	When("Given names and plan template AZURE", func() {
		It("Should pass flow", func() {
			testFlow.Login()

			By("Can create service broker from repo and setup env", func() {
				testFlow.PushBroker()
				testFlow.SetDefaultEnv()
				testFlow.SetAzureEnv()
				testFlow.RestartBrokerApp()
			})

			testFlow.CreateServiceBroker()

			By("Possible to create a service", func() {
				testFlow.CreateService()
			})

			By("Check PE status", func() {
				AC := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				Expect(AC.GetAzurePrivateEndpointStatus(testFlow)).Should(Equal("AVAILABLE"))
			})

			By("Can scale cluster size", func() {
				newSize := "M20"
				testFlow.UpgradeClusterConfig()
				testFlow.WaitServiceStatus("update succeeded")
				AC := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				Expect(AC.GetClusterSize(testFlow)).Should(Equal(newSize))
			})

			testFlow.DeleteResources()
		})
	})
})
