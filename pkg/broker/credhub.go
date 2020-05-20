package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Credentials struct {
	PublicKey   string `json:"public_key"`
	APIKey      string `json:"api_key"`
	DisplayName string `json:"display_name"`
}

type CredHub struct {
	BindingName string                 `json:"binding_name"`
	Credentials map[string]Credentials `json:"credentials"`
}

type services struct {
	CredHub []CredHub `json:"credhub"`
}

func CredHubCredentials() (map[string]Credentials, error) {
	env, found := os.LookupEnv("VCAP_SERVICES")
	if !found {
		return nil, errors.New("VCAP_SERVICES not specified - is CredHub bound to broker?")
	}

	services := &services{}
	if err := json.Unmarshal([]byte(env), services); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %v", err)
	}

	result := map[string]Credentials{}
	for _, c := range services.CredHub {
		for k, v := range c.Credentials {
			result[k] = v
		}
	}

	return result, nil
}
