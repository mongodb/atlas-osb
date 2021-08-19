package pcf

import (
	"encoding/json"
	"os"

	utils "github.com/mongodb/atlas-osb/test/cfe2e/utils"
)

type PCF struct {
	URL      string `json:"url"`
	User     string `json:"username"`
	Password string `json:"password"`
}

func CreatePCF() (PCF, error) {
	PCFKeys := PCF{
		URL:      os.Getenv("INPUT_CF_URL"),
		User:     os.Getenv("INPUT_CF_USER"),
		Password: os.Getenv("INPUT_CF_PASSWORD"),
	}
	err := PCFKeys.createMetadata()
	if err != nil {
		return PCF{}, err
	}
	return PCFKeys, nil
}

func (pcf *PCF) createMetadata() error {
	type opsmgr struct {
		Opsmgr PCF `json:"opsmgr"`
	}
	ops := opsmgr{Opsmgr: *pcf}
	data, err := json.Marshal(ops)
	if err != nil {
		return err
	}
	err = utils.SaveToFile("metadata", data)
	if err != nil {
		return err
	}
	return nil
}
