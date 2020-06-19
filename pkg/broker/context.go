package broker

import (
	"context"
	"errors"

	"github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
)

// ContextKey represents the key for a value saved in a context. Linter
// requires keys to have their own type.
type contextKey string

// ContextKeyAtlasClient is the key used to store the Atlas client in the
// request context.
var ContextKeyAtlasClient = contextKey("atlas-client")

// atlasClientFromContext will retrieve an Atlas client stored inside the
// provided context.
func atlasClientFromContext(ctx context.Context) (atlas.Client, error) {
	client, ok := ctx.Value(ContextKeyAtlasClient).(atlas.Client)
	if !ok {
		return nil, errors.New("no Atlas client in context")
	}

	return client, nil
}
