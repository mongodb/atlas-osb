package dynamicplans

import "github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"

type Context struct {
	Project mongodbatlas.Project
	Cluster mongodbatlas.Cluster
}

func DefaultCtx() Context {
	return Context{}
}
