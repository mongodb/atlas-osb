package cf

import (
	"fmt"
	"regexp"

	"github.com/mongodb/atlas-osb/test/cfe2e/cli/pcf"
	pcfmeta "github.com/mongodb/atlas-osb/test/cfe2e/model/pcf"
)

type CF struct { // Application
	User     string
	Password string
	URL      string
}

// get access to cloudfoundry
func NewCF() CF {
	pcfmeta.CreatePCF()
	output := string(pcf.Info())
	CF := CF{}
	CF.Password = regexp.MustCompile("- admin_password: (.+)").FindStringSubmatch(output)[1]
	CF.User = regexp.MustCompile("- admin_username: (.+)").FindStringSubmatch(output)[1]
	CF.URL = fmt.Sprintf("http://api.%s", regexp.MustCompile("- system_domain: (.+)").FindStringSubmatch(output)[1])
	return CF
}
