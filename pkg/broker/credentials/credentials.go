package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// FIXME: temporary hack for old secrets
type APIKey struct {
	mongodbatlas.APIKey
	DisplayName string `json:"display_name,omitempty"`
}

type Credentials struct {
	Projects map[string]APIKey `json:"projects"`
	Orgs     map[string]APIKey `json:"orgs"`
	Broker   *BrokerAuth       `json:"broker"`
}

type BrokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DB       string `json:"db"`
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
		return nil, nil
	}

	services := &services{}
	if err := json.Unmarshal([]byte(env), services); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %v", err)
	}

	result := Credentials{
		Projects: map[string]APIKey{},
		Orgs:     map[string]APIKey{},
	}

	for _, c := range services.CredHub {
		for k, v := range c.Credentials.Projects {
			// FIXME: temporary hack for old secrets
			if v.DisplayName != "" {
				v.Desc = v.DisplayName
			}

			result.Projects[k] = v
		}
		for k, v := range c.Credentials.Orgs {
			result.Orgs[k] = v
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
		return nil, nil
	}

	creds := Credentials{}
	if err := json.Unmarshal([]byte(env), &creds); err != nil {
		return nil, fmt.Errorf("cannot unmarshal BROKER_APIKEYS: %v", err)
	}

	if err := creds.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %v", err)
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

	return nil
}

func (c *Credentials) FlattenOrgs(baseURL string) error {
	for k, v := range c.Orgs {
		hc, err := digest.NewTransport(v.PublicKey, v.PrivateKey).Client()
		if err != nil {
			return err
		}

		client, err := mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
		if err != nil {
			return err
		}

		p, _, err := client.Projects.GetAllProjects(context.Background(), nil)
		if err != nil {
			return err
		}
		for _, pp := range p.Results {
			if pp.OrgID != k {
				continue
			}
			c.Projects[pp.ID] = v
		}
	}
	c.Orgs = map[string]APIKey{}
	return nil
}
