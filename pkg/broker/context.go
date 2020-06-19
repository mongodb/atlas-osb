package broker

import (
	"context"
	"errors"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// ContextKey represents the key for a value saved in a context. Linter
// requires keys to have their own type.
type contextKey string

// ContextKeyAtlasClient is the key used to store the Atlas client in the
// request context.
const (
	ContextKeyAtlasClient contextKey = "atlas-client"
	ContextKeyGroupID     contextKey = "group-id"
)

// atlasClientFromContext will retrieve an Atlas client stored inside the
// provided context.
func atlasClientFromContext(ctx context.Context) (*mongodbatlas.Client, error) {
	client, ok := ctx.Value(ContextKeyAtlasClient).(*mongodbatlas.Client)
	if !ok {
		return nil, errors.New("no Atlas client in context")
	}

	return client, nil
}

func groupIDFromContext(ctx context.Context) (string, error) {
	gid, ok := ctx.Value(ContextKeyGroupID).(string)
	if !ok {
		return "", errors.New("no group ID in context")
	}

	return gid, nil
}
