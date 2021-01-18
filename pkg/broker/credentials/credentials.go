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
	"go.mongodb.org/atlas/mongodbatlas"
)

type BrokerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// DB       string `json:"db"`
}

type Credentials struct {
	aliases map[string]string
	byOrg   map[string]APIKey
	Broker  *BrokerAuth
}

type keyList struct {
	Keys   map[string]APIKey `json:"keys"`
	Broker *BrokerAuth       `json:"broker"`
}

type APIKey struct {
	ID         string                   `json:"id,omitempty"`
	Desc       string                   `json:"desc,omitempty"`
	Roles      []mongodbatlas.AtlasRole `json:"roles,omitempty"`
	PrivateKey string                   `json:"privateKey,omitempty"`
	PublicKey  string                   `json:"publicKey,omitempty"`
	OrgID      string                   `json:"orgID,omitempty"`
}

func (k APIKey) MongoKey() mongodbatlas.APIKey {
	return mongodbatlas.APIKey{
		ID:         k.ID,
		Desc:       k.Desc,
		Roles:      k.Roles,
		PrivateKey: k.PrivateKey,
		PublicKey:  k.PublicKey,
	}
}

type credHub struct {
	BindingName string  `json:"binding_name"`
	KeyList     keyList `json:"credentials"`
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
		return nil, fmt.Errorf("cannot unmarshal VCAP_SERVICES: %w", err)
	}

	result := Credentials{
		aliases: map[string]string{},
		byOrg:   map[string]APIKey{},
	}

	for _, c := range append(services.CredHub, services.UserProvided...) {
		for k, v := range c.KeyList.Keys {
			result.aliases[k] = v.OrgID
			result.byOrg[v.OrgID] = v
		}

		if c.KeyList.Broker != nil {
			result.Broker = c.KeyList.Broker
		}
	}

	if err := result.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %w", err)
	}

	return &result, nil
}

func FromEnv(baseURL string) (*Credentials, error) {
	env, found := os.LookupEnv("BROKER_APIKEYS")
	if !found {
		return nil, nil
	}

	keys := keyList{
		Keys:   map[string]APIKey{},
		Broker: &BrokerAuth{},
	}

	if err := json.Unmarshal([]byte(env), &keys); err != nil {
		file, err := os.Open(env)
		if err != nil {
			return nil, fmt.Errorf("cannot find BROKER_APIKEYS: %w", err)
		}
		defer file.Close()

		fileData, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("cannot read BROKER_APIKEYS: %w", err)
		}
		if err := json.Unmarshal(fileData, &keys); err != nil {
			return nil, fmt.Errorf("cannot unmarshal BROKER_APIKEYS: %w", err)
		}
	}

	result := Credentials{
		aliases: map[string]string{},
		byOrg:   map[string]APIKey{},
		Broker:  keys.Broker,
	}

	for k, v := range keys.Keys {
		result.aliases[k] = v.OrgID
		result.byOrg[v.OrgID] = v
	}

	if err := result.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %w", err)
	}

	return &result, nil
}

func (c *Credentials) validate() error {
	if c.Broker == nil {
		return errors.New("no broker credentials specified")
	}

	if len(c.byOrg) == 0 {
		return errors.New("no API keys specified")
	}

	return nil
}

func (c *Credentials) ByAlias(alias string) (APIKey, error) {
	id, ok := c.aliases[alias]
	if !ok {
		return APIKey{}, fmt.Errorf("no organization ID for alias %q", alias)
	}

	k, ok := c.byOrg[id]
	if !ok {
		return k, fmt.Errorf("no API key for organization %s", id)
	}

	return k, nil
}

func (c *Credentials) ByOrg(id string) (APIKey, error) {
	k, ok := c.byOrg[id]
	if !ok {
		return k, fmt.Errorf("no API key for organization %s", id)
	}

	return k, nil
}

func (c *Credentials) Keys() map[string]APIKey {
	return c.byOrg
}
