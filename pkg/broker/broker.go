package broker

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"go.uber.org/zap"
)

// Ensure broker adheres to the ServiceBroker interface.
var _ brokerapi.ServiceBroker = Broker{}

// Broker is responsible for translating OSB calls to Atlas API calls.
// Implements the brokerapi.ServiceBroker interface making it easy to spin up
// an API server.
type Broker struct {
	logger    *zap.SugaredLogger
	whitelist Whitelist
	credHub   map[string]Credentials
	baseURL   string
}

// NewBroker creates a new Broker with a logger.
func NewBroker(logger *zap.SugaredLogger, credHub map[string]Credentials, baseURL string) *Broker {
	return &Broker{
		logger:  logger,
		credHub: credHub,
		baseURL: baseURL,
	}
}

// NewBrokerWithWhitelist creates a new Broker with a given logger and a
// whitelist for allowed providers and their plans.
func NewBrokerWithWhitelist(logger *zap.SugaredLogger, credHub map[string]Credentials, baseURL string, whitelist Whitelist) *Broker {
	return &Broker{
		logger:    logger,
		credHub:   credHub,
		baseURL:   baseURL,
		whitelist: whitelist,
	}
}

// ContextKey represents the key for a value saved in a context. Linter
// requires keys to have their own type.
type ContextKey string

// ContextKeyAtlasClient is the key used to store the Atlas client in the
// request context.
var ContextKeyAtlasClient = ContextKey("atlas-client")

// AuthMiddleware is used to validate and parse Atlas API credentials passed
// using basic auth. The credentials parsed into an Atlas client which is
// attached to the request context. This client can later be retrieved by the
// broker from the context.
func AuthMiddleware(credhub map[string]Credentials) mux.MiddlewareFunc {
	bc := credhub["broker"]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if bc.PublicKey != username || bc.APIKey != password {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// atlasToAPIError converts an Atlas error to a OSB response error.
func atlasToAPIError(err error) error {
	switch err {
	case atlas.ErrClusterNotFound:
		return apiresponses.ErrInstanceDoesNotExist
	case atlas.ErrClusterAlreadyExists:
		return apiresponses.ErrInstanceAlreadyExists
	case atlas.ErrUserAlreadyExists:
		return apiresponses.ErrBindingAlreadyExists
	case atlas.ErrUserNotFound:
		return apiresponses.ErrBindingDoesNotExist
	case atlas.ErrUnauthorized:
		return apiresponses.NewFailureResponse(err, http.StatusUnauthorized, "")
	}

	// Fall back on returning the error again if no others match.
	// Will result in a 500 Internal Server Error.
	return err
}
