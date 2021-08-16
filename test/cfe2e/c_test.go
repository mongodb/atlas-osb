package cfe2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"

	cfc "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var _ = Describe("Feature: Atlas broker supports basic template", func() {

	When("Given names and plan template", func() {
		It("Should pass flow", func() {
			By("Can login to CF and create organization", func() {
				Eventually(cfc.Cf("login", "-a", PCFKeys.Endpoint, "-u", PCFKeys.User, "-p", PCFKeys.Password, "--skip-ssl-validation")).Should(Say("OK"))
				Eventually(cfc.Cf("create-org", orgName)).Should(Say("OK"))
				Eventually(cfc.Cf("target", "-o", orgName)).Should(Exit(0))
				Eventually(cfc.Cf("create-space", spaceName)).Should(Exit(0))
				Eventually(cfc.Cf("target", "-s", spaceName)).Should(Exit(0))
			})
			By("Can create service broker from repo and setup env", func() {
				s := cfc.Cf("push", brokerApp, "-p", "../../.", "--no-start") //ginkgo starts from test-root folder
				Eventually(s, "2m", "10s").Should(Exit(0))                    //TODO probably one of the common timeouts
				Eventually(s).Should(Say("down"))
				Eventually(cfc.Cf("set-env", brokerApp, "BROKER_HOST", "0.0.0.0")).Should(Exit(0))
				Eventually(cfc.Cf("set-env", brokerApp, "BROKER_PORT", "8080")).Should(Exit(0))
				cKey, _ := json.Marshal(APIKeys)
				Eventually(cfc.Cf("set-env", brokerApp, "BROKER_APIKEYS", string(cKey))).Should(Exit(0))
				Eventually(cfc.Cf("set-env", brokerApp, "ATLAS_BROKER_TEMPLATEDIR", tPath)).Should(Exit(0))
				Eventually(cfc.Cf("set-env", brokerApp, "BROKER_OSB_SERVICE_NAME", mPlaceName)).Should(Exit(0))

				s = cfc.Cf("restart", brokerApp)
				Eventually(s, "5m", "10s").Should(Say("running")) //TODO probably one of the common timeouts
				brokerURL = "http://" + string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
			})
			By("Possible to create service-broker", func() {
				GinkgoWriter.Write([]byte(brokerURL))
				Eventually(cfc.Cf("create-service-broker", broker, APIKeys.Broker.Username, APIKeys.Broker.Password, brokerURL, "--space-scoped")).Should(Exit(0))
				Eventually(cfc.Cf("marketplace")).Should(Say(mPlaceName))
			})

			By("Possible to create a service", func() {
				orgID := APIKeys.Keys["TKey"]["OrgID"]
				c := fmt.Sprintf("{\"org_id\":\"%s\"}", orgID)
				s := cfc.Cf("create-service", mPlaceName, planName, serviceIns, "-c", c)
				Eventually(s).Should(Exit(0))
				Eventually(s).Should(Say("OK"))
				Eventually(s).ShouldNot(Say("Service instance already exists"))
				waitServiceStatus(serviceIns, "create succeeded")
			})

			//TODO: PARALLEL CHECKS
			By("Can install test app", func() {
				testAppRepo := "https://github.com/leo-ri/simple-ruby.git"
				_, err := git.PlainClone("simple-ruby", false, &git.CloneOptions{ //TODO change with mini-docker image
					URL:               testAppRepo,
					RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
				})
				if err != nil {
					GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get test application %s", appURL)))
				}

				Eventually(cfc.Cf("push", testApp, "-p", "./simple-ruby", "--no-start")).Should(Say("down"))
				Eventually(cfc.Cf("bind-service", testApp, serviceIns)).Should(Say("OK"))

				s := cfc.Cf("restart", testApp)
				Eventually(s, "2m", "10s").Should(Exit(0))
				Eventually(s, "2m", "10s").Should(Say("running"))
				appURL = string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
				Expect(appURL).ShouldNot(BeEmpty())
			})
			By("Can send data to cluster and get it back", func() {
				// appURL = "simple-ruby.apps.sanmarcos.cf-app.com"
				URL := fmt.Sprintf("http://%s/service/mongo/test", appURL)
				ds := `{"data":"somesimpletest130"}` //TODO gen

				respCode := putData(URL, ds)
				Expect(respCode).Should(Equal(200))

				respCode, answer := getData(URL)
				Expect(respCode).Should(Equal(200))
				Expect(answer).To(Equal(ds))
			})
			By("Possible to create service-key", func() {
				Eventually(cfc.Cf("create-service-key", serviceIns, "atlasKey")).Should(Say("OK"))
				// '{"user" : { "roles" : [ { "roleName":"atlasAdmin", "databaseName" : "admin" } ] } }'
				GinkgoWriter.Write([]byte("Possible to create service-key. Check is not ready")) //TODO !
			})
			By("Backup is active as default", func() {
				path := fmt.Sprintf("data/%s.yml.tpl", planName)
				backup := getBackupStateFromPlanConfig(path)

				AC := AClient()
				projectInfo, _, _ := AC.Projects.GetOneProjectByName(context.Background(), serviceIns)
				clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, serviceIns)
				Expect(clusterInfo.ProviderBackupEnabled).To(PointTo(Equal(backup)))
			})
			By("Can scale cluster size", func() {
				newSize := "M20"
				Eventually(cfc.Cf("update-service", serviceIns, "-c", "{\"instance_size\":\"M20\"}")).Should(Say("OK"))
				waitServiceStatus(serviceIns, "update succeeded")

				// get the real size
				AC := AClient()
				projectInfo, _, err := AC.Projects.GetOneProjectByName(context.Background(), serviceIns)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(projectInfo.ID).ShouldNot(BeEmpty())
				clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, serviceIns)
				Expect(clusterInfo.ProviderSettings.InstanceSizeName).Should(Equal(newSize))
			})
			By("Possible to continue using app after update", func() {
				// appURL = "simple-app-serial-tests-bc0f0d9-2.apps.sanmarcos.cf-app.com"
				URL := fmt.Sprintf("http://%s/service/mongo/test", appURL)
				ds := `{"data":"somesimpletest130"}` //TODO gen
				respCode, answer := getData(URL)
				Expect(respCode).Should(Equal(200))
				Expect(answer).To(Equal(ds))
			})
			By("Possible to PUT new data after update", func() {
				// appURL = "simple-app-serial-tests-bc0f0d9-2.apps.sanmarcos.cf-app.com"
				URL := fmt.Sprintf("http://%s/service/mongo/test2", appURL)
				ds := `{"data":"somesimpletest130update"}` //TODO gen

				respCode := putData(URL, ds)
				Expect(respCode).Should(Equal(200))

				respCode, answer := getData(URL)
				Expect(respCode).Should(Equal(200))
				Expect(answer).To(Equal(ds))
			})
			//TODO move to tierdown
			By("Possible to delete service-key", func() {
				Eventually(cfc.Cf("delete-service-key", serviceIns, "atlasKey", "-f")).Should(Say("OK"))
			})
			By("Possible to unbind service", func() {
				Eventually(cfc.Cf("unbind-service", testApp, serviceIns)).Should(Say("OK"))
			})
			By("Possible to delete test application after use", func() {
				Eventually(cfc.Cf("delete", testApp, "-f")).Should(Say("OK"))
			})
			By("Possible to delete service", func() {
				Eventually(cfc.Cf("delete-service", serviceIns, "-f")).Should(Say("OK"))
				waitForDelete()
			})
			By("Possible to delete Service broker", func() {
				Eventually(cfc.Cf("delete-service-broker", broker, "-f")).Should(Say("OK"))
			})
			By("Possible to delete broker application", func() {
				Eventually(cfc.Cf("delete", brokerApp, "-f")).Should(Say("OK"))
			})
		})

	})
})

func waitForDelete() {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) //TODO
		try++
		session := cfc.Cf("services")
		EventuallyWithOffset(1, session).Should(Exit(0))
		isDeleted := strings.Contains(string(session.Out.Contents()), "No services found")
		GinkgoWriter.Write([]byte(fmt.Sprintf("Waiting for deletion (try #%d)", try)))

		if isDeleted {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 13 { //TODO what is our req. for awaiting
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			ExpectWithOffset(1, true).Should(Equal(false)) //TODO call fail
		}
	}
}

//waitStatus wait until status is appear
func waitServiceStatus(serviceName string, expectedStatus string) {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) //TODO :\\
		try++
		s := cfc.Cf("service", serviceName)
		EventuallyWithOffset(1, s).Should(Exit(0))
		status := string(regexp.MustCompile(`status:\s+(.+)\s+`).FindSubmatch(s.Out.Contents())[1])
		GinkgoWriter.Write([]byte(fmt.Sprintf("Status is %s (try #%d)", status, try)))
		if status == expectedStatus {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 15 { //TODO ??
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			ExpectWithOffset(1, true).Should(Equal(false)) //TODO call fail
		}
	}
}

func putData(appURL string, ds string) int {
	r, err := http.NewRequest("PUT", appURL, strings.NewReader(ds))
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	client := &http.Client{}

	resp, err := client.Do(r)
	if err != nil {
		GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get response %s", err)))
	}
	defer resp.Body.Close()
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	return resp.StatusCode
}

func getData(appURL string) (int, string) {
	resp, err := http.Get(appURL) //nolint
	if err != nil {
		GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get response %s", err)))
	}
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	defer resp.Body.Close()
	responseData, _ := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, string(responseData)
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
