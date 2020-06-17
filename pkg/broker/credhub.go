package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Credential struct {
	PublicKey   string `json:"public_key"`
	APIKey      string `json:"api_key"`
	DisplayName string `json:"display_name"`
}

type brokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type credentials struct {
	Projects map[string]Credential `json:"projects"`
	Orgs     map[string]Credential `json:"orgs"`
	Broker   *brokerAuth           `json:"broker"`
}

type credHub struct {
	BindingName string      `json:"binding_name"`
	Credentials credentials `json:"credentials"`
}

type services struct {
	CredHub []credHub `json:"credhub"`
}

func CredHubCredentials() (*credentials, error) {
	env, found := os.LookupEnv("VCAP_SERVICES")
	if !found {
		return nil, fmt.Errorf("env VCAP_SERVICES not specified")
	}

	services := &services{}
	if err := json.Unmarshal([]byte(env), services); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %v", err)
	}

	result := credentials{
		Projects: map[string]Credential{},
		Orgs:     map[string]Credential{},
	}

	for _, c := range services.CredHub {
		for k, v := range c.Credentials.Projects {
			result.Projects[k] = v
		}
		for k, v := range c.Credentials.Orgs {
			result.Projects[k] = v
		}
		if c.Credentials.Broker != nil {
			result.Broker = c.Credentials.Broker
		}
	}

	if err := result.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %v", err)
	}

	return &result, nil
}

func EnvCredentials() (*credentials, error) {
	env, found := os.LookupEnv("BROKER_APIKEYS")
	if !found {
		return nil, fmt.Errorf("env BROKER_APIKEYS not specified")
	}

	creds := credentials{}
	if err := json.Unmarshal([]byte(env), &creds); err != nil {
		return nil, fmt.Errorf("cannot unmarshal BROKER_APIKEYS: %v", err)
	}

	if err := creds.validate(); err != nil {
		return nil, err
	}

	return &creds, nil
}

func (c credentials) validate() error {
	if c.Broker == nil {
		return errors.New("no broker credentials specified")
	}

	if len(c.Projects)+len(c.Orgs) == 0 {
		return errors.New("no Project/Org credentials specified")
	}

	if len(c.Projects) == 0 && len(c.Orgs) != 0 {
		return errors.New("Org credentials are not implemented yet")
	}

	return nil
}
