package dynamicplans

import "github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"

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
}

// Binding info
type Binding struct {
}
