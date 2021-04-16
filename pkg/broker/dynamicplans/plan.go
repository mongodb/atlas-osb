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

package dynamicplans

import (
	"encoding/json"

	"github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"go.mongodb.org/atlas/mongodbatlas"
)

// Plan represents a set of MongoDB Atlas resources
type Plan struct {
	Version       string                                `json:"version,omitempty"`
	Name          string                                `json:"name,omitempty"`
	Description   string                                `json:"description,omitempty"`
	Free          *bool                                 `json:"free,omitempty"`
	APIKey        credentials.Credential                `json:"apiKey,omitempty"`
	Project       *mongodbatlas.Project                 `json:"project,omitempty"`
	Cluster       *mongodbatlas.Cluster                 `json:"cluster,omitempty"`
	DatabaseUsers []*mongodbatlas.DatabaseUser          `json:"databaseUsers,omitempty"`
	IPAccessLists []*mongodbatlas.ProjectIPAccessList   `json:"ipAccessLists,omitempty"`
	Integrations  []*mongodbatlas.ThirdPartyIntegration `json:"integrations,omitempty"`

	Settings map[string]interface{} `json:"settings,omitempty"`

	// Deprecated: Use IPAccessLists instead!
	IPWhitelists []*mongodbatlas.ProjectIPWhitelist `json:"ipWhitelists,omitempty"`
}

func (p *Plan) SafeCopy() Plan {
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	safe := Plan{}
	err = json.Unmarshal(b, &safe)
	if err != nil {
		panic(err)
	}

	if safe.APIKey != nil && safe.APIKey["privateKey"] != "" {
		safe.APIKey["privateKey"] = "*REDACTED*"
	}

	for i := range safe.DatabaseUsers {
		if safe.DatabaseUsers[i].Password != "" {
			safe.DatabaseUsers[i].Password = "*REDACTED*"
		}
	}

	return safe
}

func (p Plan) String() string {
	s, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(s)
}
