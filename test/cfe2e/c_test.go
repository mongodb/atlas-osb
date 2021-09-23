package cfe2e

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/go-git/go-git/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"

	"github.com/mongodb/atlas-osb/test/cfe2e/model/cf"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
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
			s := cfc.Cf("logs", testFlow.BrokerApp, "--recent")
			Expect(s).Should(Exit(0))
			utils.SaveToFile(fmt.Sprintf("output/%s", testFlow.BrokerApp), s.Out.Contents())
			testFlow.DeleteApplicationResources()
			testFlow.DeleteResources()
		}
	})

	When("Given names and plan template", func() {
		It("Should pass flow", func() {
			By("Can login to CF and create organization", func() {
				PCFKeys := cf.NewCF()
				Expect(PCFKeys).To(MatchFields(IgnoreExtras, Fields{
					"URL":      Not(BeEmpty()),
					"User":     Not(BeEmpty()),
					"Password": Not(BeEmpty()),
				}))
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

			// TODO: PARALLEL CHECKS
			By("Can install test app", func() {
				testAppRepo := "https://github.com/leo-ri/simple-ruby.git"
				_, err := git.PlainClone("simple-ruby", false, &git.CloneOptions{
					URL:               testAppRepo,
					RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
				})
				if err != nil {
					GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get test application %s", testFlow.AppURL)))
				}
				Eventually(cfc.Cf("push", testFlow.TestApp, "-p", "./simple-ruby", "--no-start")).Should(Say("down"))
				Eventually(cfc.Cf("bind-service", testFlow.TestApp, testFlow.ServiceIns)).Should(Say("OK"))

				s := cfc.Cf("restart", testFlow.TestApp)
				Eventually(s, CFEventuallyTimeoutMiddle, IntervalMiddle).Should(Exit(0))
				Eventually(s, CFEventuallyTimeoutMiddle, IntervalMiddle).Should(Say("running"))
				Eventually(cfc.Cf("app", testFlow.TestApp), "5m", IntervalMiddle).Should(Say("running"))
				testFlow.AppURL = string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
				Expect(testFlow.AppURL).ShouldNot(BeEmpty())
			})
			By("Can send data to cluster and get it back", func() {
				data := `{"data":"somesimpletest130"}` // TODO gen

				app := apptest.NewTestAppClient("http://" + testFlow.AppURL)
				Expect(app.Get("")).Should(Equal("hello from sinatra"))
				Expect(app.PutData("/service/mongo/test", data)).ShouldNot(HaveOccurred())
				Expect(app.Get("/service/mongo/test")).Should(Equal(data))
			})
			By("Possible to create service-key", func() {
				Eventually(cfc.Cf("create-service-key", testFlow.ServiceIns, "atlasKey")).Should(Say("OK"))
				// '{"user" : { "roles" : [ { "roleName":"atlasAdmin", "databaseName" : "admin" } ] } }'
				GinkgoWriter.Write([]byte("Possible to create service-key. Check is not ready")) // TODO !
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
