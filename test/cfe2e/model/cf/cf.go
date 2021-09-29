package cf

import (
	"errors"
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
func NewCF() (CF, error) {
	pcfmeta.CreatePCF()
	output := string(pcf.Info())
	CF := CF{}

	pass := regexp.MustCompile("- admin_password: (.+)").FindStringSubmatch(output)
	if len(pass) != 2 {
		return CF, errors.New("can't find admin_password")
	}
	CF.Password = pass[1]

	user := regexp.MustCompile("- admin_username: (.+)").FindStringSubmatch(output)
	if len(pass) != 2 {
		return CF, errors.New("can't find admin_username")
	}
	CF.User = user[1]

	url := regexp.MustCompile("- system_domain: (.+)").FindStringSubmatch(output)
	if len(pass) != 2 {
		return CF, errors.New("can't find system_domain")
	}
	CF.URL = fmt.Sprintf("http://api.%s", url[1])

	return CF, nil
}
