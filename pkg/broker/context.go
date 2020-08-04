package broker

import (
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

