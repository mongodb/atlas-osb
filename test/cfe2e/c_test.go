package cfe2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gstruct"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	apptest "github.com/mongodb/atlas-osb/test/cfe2e/utils/apptest"
	cfc "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var _ = Describe("Feature: Atlas broker supports basic template[standart-flow]", func() {
	var testFlow test.Test

	_ = BeforeEach(func() {
		By("Set up", func() {
			testFlow = test.NewTest()
			Expect(testFlow.APIKeys.Keys[TKey]).Should(HaveKeyWithValue("publicKey", Not(BeEmpty())))
		})
	})

	_ = AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			testFlow.SaveLogs()
			testFlow.DeleteApplicationResources()
			testFlow.DeleteResources()
		}
	})

	When("Given names and plan template", func() {
		It("Should pass flow", func() {
			testFlow.Login()

			By("Can create service broker from repo and setup env", func() {
				testFlow.PushBroker()
				testFlow.SetDefaultEnv()
				testFlow.RestartBrokerApp()
			})
			By("Possible to create service-broker", func() {
				testFlow.CreateServiceBroker()
			})

			By("Possible to create a service", func() {
				testFlow.CreateService()
				testFlow.WaitServiceStatus("create succeeded")
			})

			By("Can install test app", func() {
				testFlow.PushTestAppAndBindService()
			})
			By("Can send data to cluster and get it back", func() {
				data := `{"data":"somesimpletest130"}` // TODO gen

				app := apptest.NewTestAppClient("http://" + testFlow.AppURL)
				Expect(app.Get("")).Should(Equal("hello from sinatra"))
				Expect(app.PutData("/service/mongo/test", data)).ShouldNot(HaveOccurred())
				Expect(app.Get("/service/mongo/test")).Should(Equal(data))
			})
			By("Possible to create service-key", func() {
				testFlow.CreateServiceKey()
			})
			By("Backup is active as default", func() {
				path := fmt.Sprintf("data/%s.yml.tpl", testFlow.PlanName)
				backup := getBackupStateFromPlanConfig(path)
				AC := AClient(testFlow.APIKeys.Keys[TKey])
				projectInfo, _, _ := AC.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
				clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, testFlow.ServiceIns)
				Expect(clusterInfo.ProviderBackupEnabled).To(PointTo(Equal(backup)))
			})
			By("Can scale cluster size", func() {
				newSize := "M20"
				Eventually(cfc.Cf("update-service", testFlow.ServiceIns, "-c", testFlow.UpdateType)).Should(Say("OK"))
				testFlow.WaitServiceStatus("update succeeded")

				// get the real size
				AC := AClient(testFlow.APIKeys.Keys[TKey])
				projectInfo, _, err := AC.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(projectInfo.ID).ShouldNot(BeEmpty())
				clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, testFlow.ServiceIns)
				Expect(clusterInfo.ProviderSettings.InstanceSizeName).Should(Equal(newSize))
			})
			By("Possible to continue using app after update", func() {
				data := `{"data":"somesimpletest130"}` // TODO gen
				app := apptest.NewTestAppClient("http://" + testFlow.AppURL)
				Expect(app.Get("/service/mongo/test")).Should(Equal(data))
			})
			By("Possible to PUT new data after update", func() {
				data := `{"data":"somesimpletest130update"}` // TODO gen

				app := apptest.NewTestAppClient("http://" + testFlow.AppURL)
				Expect(app.PutData("/service/mongo/test2", data)).ShouldNot(HaveOccurred())
				Expect(app.Get("/service/mongo/test2")).Should(Equal(data))
			})

			testFlow.DeleteApplicationResources()
			testFlow.DeleteResources()
		})
	})
})
