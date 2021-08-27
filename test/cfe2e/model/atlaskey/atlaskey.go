package atlaskey

import (
	"os"

	c "github.com/mongodb/atlas-osb/pkg/broker/credentials"
)

type KeyList struct {
	Keys   map[string]c.Credential `json:"keys"`
	Broker *c.BrokerAuth           `json:"broker"`
}

func NewAtlasKeys() KeyList {
	keys := c.Credential{
		"orgID":      os.Getenv("INPUT_ATLAS_ORG_ID"),
		"publicKey":  os.Getenv("INPUT_ATLAS_PUBLIC_KEY"),
		"privateKey": os.Getenv("INPUT_ATLAS_PRIVATE_KEY"),
	}
	APIKeys := KeyList{
		Keys: map[string]c.Credential{
			"testKey": keys,
		},
		Broker: &c.BrokerAuth{
			Username: "admin",
			Password: "admin",
		},
	}
	return APIKeys
}
