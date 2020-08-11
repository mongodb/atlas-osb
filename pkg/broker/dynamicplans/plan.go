package dynamicplans

import (
	"encoding/json"

	"github.com/jinzhu/copier"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// Plan represents a set of MongoDB Atlas resources
type Plan struct {
	Version             string                             `json:"version,omitempty"`
	Name                string                             `json:"name,omitempty"`
	Description         string                             `json:"description,omitempty"`
	Free                *bool                              `json:"free,omitempty"`
	APIKey              *mongodbatlas.APIKey               `json:"apiKey,omitempty"`
	Project             *mongodbatlas.Project              `json:"project,omitempty"`
	Cluster             *mongodbatlas.Cluster              `json:"cluster,omitempty"`
	DatabaseUsers       []*mongodbatlas.DatabaseUser       `json:"databaseUsers,omitempty"`
	IPWhitelists        []*mongodbatlas.ProjectIPWhitelist `json:"ipWhitelists,omitempty"`
	DefaultBindingRoles *[]mongodbatlas.Role               `json:"defaultBindingRoles"`
	Bindings            []*Binding                         `json:"bindings,omitempty"` // READ ONLY! Populated by bind()

	Settings map[string]string `json:"settings,omitempty"`
}

func (p *Plan) SafeCopy() Plan {
	safe := Plan{}
	err := copier.Copy(&safe, p)
	if err != nil {
		panic(err)
	}

	if safe.APIKey != nil && safe.APIKey.PrivateKey != "" {
		safe.APIKey.PrivateKey = "*REDACTED*"
	}

	for i := range safe.DatabaseUsers {
		if safe.DatabaseUsers[i].Password != "" {
			safe.DatabaseUsers[i].Password = "*REDACTED*"
		}
	}

	return safe
}

func (p Plan) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

// Binding info
type Binding struct {
}
