package cfe2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.mongodb.org/atlas/mongodbatlas"

	"github.com/mongodb/atlas-osb/test/cfe2e/config"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/atlasclient"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	"github.com/mongodb/atlas-osb/test/cfe2e/utils"
	apptest "github.com/mongodb/atlas-osb/test/cfe2e/utils/apptest"
)

var _ = Describe("Feature: Atlas broker supports basic template[standart-flow]", func() {
	var testFlow test.Test

	_ = BeforeEach(func() {
		By("Set up", func() {
			testFlow = test.NewTest()
			Expect(testFlow.APIKeys.Keys[config.TKey]).Should(HaveKeyWithValue("publicKey", Not(BeEmpty())))
		})
	})

	_ = AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			testFlow.SaveLogs()
			testFlow.DeleteServiceKeys()
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
				role := mongodbatlas.Role{
					DatabaseName: "admin",
					RoleName:     "readAnyDatabase",
				}
				labelValue := "test-service-key"
				labels := fmt.Sprintf(", \"labels\": [{ \"key\": \"%s\", \"value\": \"%s\" }]", "name", labelValue)
				c := fmt.Sprintf("{\"user\" : {\"roles\" : [ { \"roleName\" : \"%s\", \"databaseName\" : \"%s\"} ] %s } }",
					role.RoleName, role.DatabaseName, labels,
				)
				testFlow.CreateServiceKey(c, labelValue)

				// check atlas
				ac := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				users := ac.GetDatabaseUsersList(testFlow)
				user := FindUserWithLabel(users, labelValue)
				Expect(user.Roles).Should(HaveLen(1))
				Expect(user.Roles).Should(ContainElement(role))
			})
			By("Possible to create default service-key", func() {
				role := getDefaultRole(testFlow)
				labelValue := "test-service-key-default"
				labels := fmt.Sprintf("{ \"labels\": [{ \"key\": \"%s\", \"value\": \"%s\" }] }", "name", labelValue)
				c := fmt.Sprintf("{\"user\" : %s }", labels)
				testFlow.CreateServiceKey(c, labelValue)

				// check atlas
				ac := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				users := ac.GetDatabaseUsersList(testFlow)
				user := FindUserWithLabel(users, labelValue)
				Expect(user.Roles).Should(HaveLen(1))
				Expect(user.Roles).Should(ContainElement(role))
			})
			By("Backup is active as default", func() {
				path := fmt.Sprintf("data/%s.yml.tpl", testFlow.PlanName)
				backup := getBackupStateFromPlanConfig(path)
				AC := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				Expect(AC.GetBackupState(testFlow)).To(Equal(backup))
			})
			By("Can scale cluster size", func() {
				newSize := "M20"
				testFlow.UpgradeClusterConfig()
				testFlow.WaitServiceStatus("update succeeded")
				AC := atlasclient.AClient(testFlow.APIKeys.Keys[config.TKey])
				Expect(AC.GetClusterSize(testFlow)).Should(Equal(newSize))
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
			testFlow.DeleteServiceKeys()
			testFlow.DeleteApplicationResources()
			testFlow.DeleteResources()
		})
	})
})

func FindUserWithLabel(users []mongodbatlas.DatabaseUser, label string) mongodbatlas.DatabaseUser {
	for _, user := range users {
		if len(user.Labels) > 0 {
			for _, l := range user.Labels {
				if l.Key == "name" && l.Value == label {
					return user
				}
			}
		}
	}
	return mongodbatlas.DatabaseUser{}
}

func getDefaultRole(testFlow test.Test) mongodbatlas.Role {
	pathToFile := "data/" + testFlow.PlanName + ".yml.tpl"
	roleFromFile, _ := utils.GetFieldFromFile(pathToFile, "overrideBindDBRole")
	dbFromFile, _ := utils.GetFieldFromFile(pathToFile, "overrideBindDB")
	role := mongodbatlas.Role{
		RoleName:     roleFromFile,
		DatabaseName: dbFromFile,
	}
	// if no override - get default
	if role.RoleName == "" {
		role.RoleName = "readWriteAnyDatabase"
	}
	if role.DatabaseName == "" {
		role.DatabaseName = "admin"
	}
	return role
}
