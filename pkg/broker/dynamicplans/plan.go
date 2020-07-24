package dynamicplans

import "github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"

const(
    BROKER_SETTING_OVERRIDE_BIND_DB      = "overrideBindDB"
    BROKER_SETTING_OVERRIDE_BIND_DB_ROLE = "overrideBindDBRole"
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

    Settings            map[string]string                 `json:"settings,omitempty"`
}

// Binding info
type Binding struct {
}
