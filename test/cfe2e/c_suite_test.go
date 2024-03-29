package cfe2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

/*
Following environment varialble should be present:
INPUT_ATLAS_ORG_ID
INPUT_ATLAS_PRIVATE_KEY
INPUT_ATLAS_PUBLIC_KEY
INPUT_CF_URL
INPUT_CF_USER
INPUT_CF_PASSWORD
These variables are copies of github secrets

These set up by pipeline (param.sh)
ORG_NAME
SPACE_NAME
BROKER
BROKER_APP
TEST_SIMPLE_APP
SERVICE_ATLAS
*/

const (
	CFEventuallyTimeoutDefault   = 120 * time.Second
	CFConsistentlyTimeoutDefault = 120 * time.Millisecond
	CFEventuallyTimeoutMiddle    = 10 * time.Minute
	IntervalMiddle               = 10 * time.Second

	// cf timouts
	CFStagingTimeout  = 15
	CFStartingTimeout = 15
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Atlas Broker")
}

var _ = BeforeSuite(func() {
	GinkgoWriter.Write([]byte("==============================Before==============================\n"))
	SetDefaultEventuallyTimeout(CFEventuallyTimeoutDefault)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeoutDefault)
	checkupCFinputs()
	setUp()
	GinkgoWriter.Write([]byte("========================End of Before==============================\n"))
})

func setUp() {
	os.Setenv("CF_STAGING_TIMEOUT", fmt.Sprint(CFStagingTimeout))
	os.Setenv("CF_STARTUP_TIMEOUT", fmt.Sprint(CFStartingTimeout))
}

func checkupCFinputs() {
	Expect(os.Getenv("INPUT_CF_URL")).ToNot(BeEmpty(), "Please, set up INPUT_CF_URL env")
	Expect(os.Getenv("INPUT_CF_USER")).ToNot(BeEmpty(), "Please, set up INPUT_CF_USER env")
	Expect(os.Getenv("INPUT_CF_PASSWORD")).ToNot(BeEmpty(), "Please, set up INPUT_CF_PASSWORD env")

	Expect(os.Getenv("ORG_NAME")).ToNot(BeEmpty(), "Please, use param.sh or set up ORG_NAME")
	Expect(os.Getenv("BROKER_APP")).ToNot(BeEmpty(), "Please, use param.sh or set up BROKER_APP")
	Expect(os.Getenv("BROKER")).ToNot(BeEmpty(), "Please, use param.sh or set up BROKER")
	Expect(os.Getenv("TEST_SIMPLE_APP")).ToNot(BeEmpty(), "Please, use param.sh or set up TEST_SIMPLE_APP")
	Expect(os.Getenv("TEST_PLAN")).ToNot(BeEmpty(), "Please, set up TEST_PLAN env, name of the plan in test/data folder")
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
