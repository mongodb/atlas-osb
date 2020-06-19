package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type credential struct {
	PublicKey   string `json:"public_key"`
	APIKey      string `json:"api_key"`
	DisplayName string `json:"display_name"`
}

type BrokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Credentials struct {
	Projects map[string]credential `json:"projects"`
	Orgs     map[string]credential `json:"orgs"`
	Broker   *BrokerAuth           `json:"broker"`
}

type credHub struct {
	BindingName string      `json:"binding_name"`
	Credentials Credentials `json:"credentials"`
}

type services struct {
	CredHub []credHub `json:"credhub"`
}

func FromCredHub() (*Credentials, error) {
	env, found := os.LookupEnv("VCAP_SERVICES")
	if !found {
		return nil, fmt.Errorf("env VCAP_SERVICES not specified")
	}

	services := &services{}
	if err := json.Unmarshal([]byte(env), services); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %v", err)
	}

	result := Credentials{
		Projects: map[string]credential{},
		Orgs:     map[string]credential{},
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

func FromEnv() (*Credentials, error) {
	env, found := os.LookupEnv("BROKER_APIKEYS")
	if !found {
		return nil, fmt.Errorf("env BROKER_APIKEYS not specified")
	}

	creds := Credentials{}
	if err := json.Unmarshal([]byte(env), &creds); err != nil {
		return nil, fmt.Errorf("cannot unmarshal BROKER_APIKEYS: %v", err)
	}

	if err := creds.validate(); err != nil {
		return nil, err
	}

	return &creds, nil
}

func (c *Credentials) validate() error {
	if c.Broker == nil {
		return errors.New("no broker credentials specified")
	}

	if len(c.Projects)+len(c.Orgs) == 0 {
		return errors.New("no Project/Org credentials specified")
	}

	if len(c.Orgs) != 0 {
		return errors.New("Org credentials are not implemented yet")
	}

	return nil
}
