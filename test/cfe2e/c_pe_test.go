package cfe2e

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// . "github.com/onsi/gomega/gstruct"
	"go.mongodb.org/atlas/mongodbatlas"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
)

var _ = Describe("Feature: Atlas broker supports basic template[pe-flow]", func() {
	var testFlow test.Test
	// var PCFKeys cf.CF

	_ = BeforeEach(func() {
		By("Check enviroment", func() {
			Expect(os.Getenv("AZURE_CLIENT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_ID env")
			Expect(os.Getenv("AZURE_CLIENT_SECRET")).ToNot(BeEmpty(), "Please, set up AZURE_CLIENT_SECRET env")
			Expect(os.Getenv("AZURE_TENANT_ID")).ToNot(BeEmpty(), "Please, set up AZURE_TENANT_ID env")
		})
		By("Set up", func() {
			testFlow = test.NewTest()
			Expect(testFlow.APIKeys.Keys[TKey]).Should(HaveKeyWithValue("publicKey", Not(BeEmpty())))
			// PCFKeys = cf.NewCF()
			// Expect(PCFKeys).To(MatchFields(IgnoreExtras, Fields{
			// 	"URL":      Not(BeEmpty()),
			// 	"User":     Not(BeEmpty()),
			// 	"Password": Not(BeEmpty()),
			// }))
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

			By("Possible to create service-key", func() {
				testFlow.CreateServiceKey()
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
