package cfe2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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
		It("Can login to CF and create organization", func() {
			// Login(endpoint, user, password)
			Eventually("true").Should(Equal("true"))
			Eventually(cfc.Cf("login", "-a", PCFKeys.Endpoint, "-u", PCFKeys.User, "-p", PCFKeys.Password, "--skip-ssl-validation")).Should(Say("OK"))
			Eventually(cfc.Cf("create-org", orgName)).Should(Say("OK"))
			Eventually(cfc.Cf("target", "-o", orgName)).Should(Exit(0))
		}, 10)
		It("Can create service broker from repo and setup env", func() {
			Eventually(cfc.Cf("create-space", spaceName)).Should(Exit(0))
			Eventually(cfc.Cf("target", "-s", spaceName)).Should(Exit(0))

			s := cfc.Cf("push", brokerApp, "-p", "../../.", "--no-start") //ginkgo starts from test-root folder
			Eventually(s, "2m", "10s").Should(Exit(0))                    //TODO probably one of the common timeouts
			Eventually(s).Should(Say("down"))
			Eventually(cfc.Cf("set-env", brokerApp, "BROKER_HOST", "0.0.0.0")).Should(Exit(0))
			Eventually(cfc.Cf("set-env", brokerApp, "BROKER_PORT", "8080")).Should(Exit(0))
			cKey, _ := json.Marshal(APIKeys)
			Eventually(cfc.Cf("set-env", brokerApp, "BROKER_APIKEYS", string(cKey))).Should(Exit(0))
			Eventually(cfc.Cf("set-env", brokerApp, "ATLAS_BROKER_TEMPLATEDIR", tPath)).Should(Exit(0))
			Eventually(cfc.Cf("set-env", brokerApp, "BROKER_OSB_SERVICE_NAME", mPlaceName)).Should(Exit(0))

			Eventually(cfc.Cf("env", brokerApp)).Should(Exit(0))
			s = cfc.Cf("restart", brokerApp)
			Eventually(s, "5m", "10s").Should(Say("running")) //TODO probably one of the common timeouts
			brokerURL = "http://" + string(regexp.MustCompile(`routes:[ ]*(.+)`).FindSubmatch(s.Out.Contents())[1])
		})
		It("Possible to create service-broker", func() {
			GinkgoWriter.Write([]byte(brokerURL))
			// brokerURL = "http://brokerapp.apps.sanmarcos.cf-app.com" //TODO remove
			Eventually(cfc.Cf("create-service-broker", broker, APIKeys.Broker.Username, APIKeys.Broker.Password, brokerURL, "--space-scoped")).Should(Exit(0))
			Eventually(cfc.Cf("marketplace")).Should(Say(mPlaceName))
		})

		It("Possible to create a service", func() {
			orgID := APIKeys.Keys[TKey].OrgID
			c := fmt.Sprintf("{\"org_id\":\"%s\"}", orgID)
			s := cfc.Cf("create-service", mPlaceName, planName, serviceIns, "-c", c)
			Eventually(s).Should(Exit(0))
			Eventually(s).Should(Say("OK"))
			Eventually(s).ShouldNot(Say("Service instance already exists"))
			waitServiceStatus(serviceIns, "create succeeded")
		})

		//TODO: PARALLEL CHECKS
		It("Can install test app", func() {
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
		It("Can send data to cluster and get it back", func() {
			// appURL = "simple-ruby.apps.sanmarcos.cf-app.com" //TODO remove
			appURL = fmt.Sprintf("http://%s/service/mongo/test", appURL)
			ds := `{"data":"somesimpletest130"}` //TODO gen
			r, err := http.NewRequest("PUT", appURL, strings.NewReader(ds))
			Expect(err).ShouldNot(HaveOccurred())
			client := &http.Client{}

			resp, err := client.Do(r)
			if err != nil {
				GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get response %s", err)))
			}
			defer resp.Body.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))

			resp, err = http.Get(appURL) //nolint
			if err != nil {
				GinkgoWriter.Write([]byte(fmt.Sprintf("Can't get response %s", err)))
			}
			defer resp.Body.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))

			responseData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			Expect(string(responseData)).To(Equal(ds))
		})
		It("Possible to create service-key", func() {
			Eventually(cfc.Cf("create-service-key", serviceIns, "atlasKey")).Should(Say("OK"))
			// '{"user" : { "roles" : [ { "roleName":"atlasAdmin", "databaseName" : "admin" } ] } }' 
			GinkgoWriter.Write([]byte("Possible to create service-key. Check is not ready")) //TODO !
		})
		It("Backup is active as default", func() {
			AC = AClient()
			projectInfo, _, _ := AC.Projects.GetOneProjectByName(context.Background(), serviceIns)
			clusterInfo, _, _ := AC.Clusters.Get(context.Background(), projectInfo.ID, serviceIns)
			// "providerBackupEnabled": false,
			Expect(clusterInfo.ProviderBackupEnabled).Should(PointTo(Equal(true))) //TODO from plan configuration
		})
		It("Can scale cluster size", func() {
			Eventually(cfc.Cf("update-service", serviceIns, "-c", "{\"instance_size\":\"M20\"}")).Should(Say("OK"))
			waitServiceStatus(serviceIns, "update succeeded")
		})
		It("Possible to continue using app after update", func() {

		})
		//TODO move to tierdown
		It("Possible to delete service-key", func() {
			Eventually(cfc.Cf("delete-service-key", serviceIns, "atlasKey", "-f")).Should(Say("OK"))
		})
		It("Possible to unbind service", func() {
			Eventually(cfc.Cf("unbind-service", testApp, serviceIns)).Should(Say("OK"))
		})
		It("Possible to delete test application after use", func() {
			Eventually(cfc.Cf("delete", testApp, "-f")).Should(Say("OK"))
		})
		It("Possible to delete service", func() {
			Eventually(cfc.Cf("delete-service", serviceIns, "-f")).Should(Say("OK"))
			waitForDelete(serviceIns)
		})
		It("Possible to delete Service broker", func() {
			Eventually(cfc.Cf("delete-service-broker", broker, "-f")).Should(Say("OK"))
		})
	})
})

func waitForDelete(serviceName string) {
	waiting := true
	try := 0
	for waiting {
		time.Sleep(1 * time.Minute) //TODO :\\
		try++
		session := cfc.Cf("service", serviceName)
		isDeleted := strings.Contains(string(session.Out.Contents()), "Service instance not found")
		GinkgoWriter.Write([]byte(fmt.Sprintf("Waiting for deletion (try #%d)", try)))

		if isDeleted {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 15 { //TODO ??
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			//TODO call fail
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
		Eventually(s).Should(Exit(0))
		// GinkgoWriter.Write(s.Out.Contents())
		status := string(regexp.MustCompile(`status:\s+(.+)\s+`).FindSubmatch(s.Out.Contents())[1])
		GinkgoWriter.Write([]byte(fmt.Sprintf("Status is %s (try #%d)", status, try)))
		if status == expectedStatus {
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Succeed."))
		}
		if try > 15 { //TODO ??
			waiting = false
			GinkgoWriter.Write([]byte("Finish waiting. Timeout"))
			//TODO call fail
		}
	}
}
