package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/pkg/errors"
)

type Credentials struct {
	projects map[string]mongodbatlas.APIKey
	Orgs     map[string]mongodbatlas.APIKey `json:"orgs"`
	Broker   *BrokerAuth                    `json:"broker"`
}

type BrokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// DB       string `json:"db"`
}

type credHub struct {
	BindingName string      `json:"binding_name"`
	Credentials Credentials `json:"credentials"`
}

type services struct {
	CredHub      []credHub `json:"credhub"`
	UserProvided []credHub `json:"user-provided"`
}

func FromCredHub(baseURL string) (*Credentials, error) {
	env, found := os.LookupEnv("VCAP_SERVICES")
	if !found {
		return nil, nil
	}

	services := &services{}
	if err := json.Unmarshal([]byte(env), services); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %v", err)
	}

	result := Credentials{
		projects: map[string]mongodbatlas.APIKey{},
		Orgs:     map[string]mongodbatlas.APIKey{},
	}

	for _, c := range append(services.CredHub, services.UserProvided...) {
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

func FromEnv(baseURL string) (*Credentials, error) {
	env, found := os.LookupEnv("BROKER_APIKEYS")
	if !found {
		return nil, nil
	}

	creds := Credentials{
		projects: map[string]mongodbatlas.APIKey{},
		Orgs:     map[string]mongodbatlas.APIKey{},
	}
	if err := json.Unmarshal([]byte(env), &creds); err != nil {
		file, err := os.Open(env)
		if err != nil {
			return nil, fmt.Errorf("cannot find BROKER_APIKEYS: %v", err)
		}
		defer file.Close()

		fileData, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("cannot read BROKER_APIKEYS: %v", err)
		}
		if err := json.Unmarshal(fileData, &creds); err != nil {
			return nil, fmt.Errorf("cannot unmarshal BROKER_APIKEYS: %v", err)
		}
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

	if len(c.Orgs) == 0 {
		return errors.New("no Org credentials specified")
	}

	return nil
}

func (c *Credentials) GetProjectKey(id string) (mongodbatlas.APIKey, error) {
	k, ok := c.projects[id]
	if !ok {
		return k, fmt.Errorf("no API key for project %s", id)
	}
	return k, nil
}

func (c *Credentials) AddProjectKey(k mongodbatlas.APIKey) {
	c.projects[k.ID] = k
}

func (c *Credentials) Client(baseURL string, k mongodbatlas.APIKey) (*mongodbatlas.Client, error) {
	hc, err := digest.NewTransport(k.PublicKey, k.PrivateKey).Client()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Digest client")
	}

	return mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
}

// TODO: should be removed on proper release?
func (c *Credentials) RandomKey() (orgID string, key mongodbatlas.APIKey) {
	for k, v := range c.Orgs {
		return k, v
	}

	return
}

func (c *Credentials) FlattenOrgs(baseURL string) error {
	for k, v := range c.Orgs {
		client, err := c.Client(baseURL, v)
		if err != nil {
			return errors.Wrap(err, "cannot create Atlas client")
		}

		p, _, err := client.Projects.GetAllProjects(context.Background(), nil)
		if err != nil {
			return err
		}

		for _, pp := range p.Results {
			if pp.OrgID != k {
				continue
			}
			c.projects[pp.ID] = v
		}
	}

	return nil
}
