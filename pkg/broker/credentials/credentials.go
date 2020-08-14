// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentials

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

type BrokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// DB       string `json:"db"`
}

type Credentials struct {
	byAlias,
	byOrg map[string]Key
	Broker *BrokerAuth
}

type credentialsDefinition struct {
	Keys   map[string]Key `json:"keys"`
	Broker *BrokerAuth    `json:"broker"`
}

type Key struct {
	OrgID      string `json:"orgID,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
}
type credHub struct {
	BindingName string                `json:"binding_name"`
	Credentials credentialsDefinition `json:"credentials"`
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
		byAlias: map[string]Key{},
		byOrg:   map[string]Key{},
	}

	for _, c := range append(services.CredHub, services.UserProvided...) {
		for k, v := range c.Credentials.Keys {
			result.byAlias[k] = v
			result.byOrg[v.OrgID] = v
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

	creds := credentialsDefinition{
		Keys:   map[string]Key{},
		Broker: &BrokerAuth{},
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

	result := Credentials{
		byAlias: map[string]Key{},
		byOrg:   map[string]Key{},
	}

	for k, v := range creds.Keys {
		result.byAlias[k] = v
		result.byOrg[v.OrgID] = v
	}

	if err := result.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %v", err)
	}

	return &result, nil
}

func (c *Credentials) validate() error {
	if c.Broker == nil {
		return errors.New("no broker credentials specified")
	}

	if len(c.byOrg) == 0 {
		return errors.New("no Org credentials specified")
	}

	return nil
}

func (c *Credentials) Alias(id string) (Key, error) {
	k, ok := c.byOrg[id]
	if !ok {
		return k, fmt.Errorf("no API key for project %s", id)
	}
	return k, nil
}

func (c *Credentials) Org(id string) (Key, error) {
	k, ok := c.byOrg[id]
	if !ok {
		return k, fmt.Errorf("no API key for project %s", id)
	}
	return k, nil
}

func (c *Credentials) Keys() map[string]Key {
	return c.byOrg
}

// TODO: should be removed on proper release?
func (c *Credentials) RandomKey() (orgID string, key Key) {
	for k, v := range c.byOrg {
		return k, v
	}

	return
}
