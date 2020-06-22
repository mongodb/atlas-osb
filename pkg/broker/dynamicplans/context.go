package dynamicplans

import "github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"

type Context struct {
	Project *mongodbatlas.Project `json:"project,omitempty"`
	Cluster *mongodbatlas.Cluster `json:"cluster,omitempty"`
	APIKey  *mongodbatlas.APIKey  `json:"apiKey,omitempty"`
}

func DefaultCtx() Context {
	return Context{
		Project: &mongodbatlas.Project{},
		Cluster: &mongodbatlas.Cluster{},
		APIKey:  &mongodbatlas.APIKey{},
	}
}
