package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/gorilla/mux"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"go.uber.org/zap"
)

// Ensure broker adheres to the ServiceBroker interface.
var _ domain.ServiceBroker = new(Broker)

// Broker is responsible for translating OSB calls to Atlas API calls.
// Implements the domain.ServiceBroker interface making it easy to spin up
// an API server.
type Broker struct {
	logger      *zap.SugaredLogger
	whitelist   Whitelist
	credentials *credentials.Credentials
	baseURL     string
	mode        Mode
	catalog     *catalog
}

// New creates a new Broker with a logger.
func New(logger *zap.SugaredLogger, credentials *credentials.Credentials, baseURL string, whitelist Whitelist, mode Mode) *Broker {
	return &Broker{
		logger:      logger,
		credentials: credentials,
		baseURL:     baseURL,
		whitelist:   whitelist,
		mode:        mode,
	}
}

func (b *Broker) getClient(ctx context.Context, planID string) (client *mongodbatlas.Client, gid string, err error) {
	client, err = atlasClientFromContext(ctx)
	if err != nil {
		gid, err = b.catalog.findGroupIDByPlanID(planID)
		if err != nil {
			return nil, gid, err
		}

		c, ok := b.credentials.Projects[gid]
		if !ok {
			return nil, gid, atlas.ErrUnauthorized
		}

		hc, err := digest.NewTransport(c.PublicKey, c.APIKey).Client()
		if err != nil {
			return nil, gid, err
		}

		// TODO: temporary hack
		baseURL := b.baseURL + "/api/atlas/v1.0/"

		client, err = mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
		if err != nil {
			return nil, gid, err
		}
	}

	gid, err = groupIDFromContext(ctx)
	return client, gid, err
}

func (b *Broker) AuthMiddleware() mux.MiddlewareFunc {
	if b.credentials != nil {
		return authMiddleware(*b.credentials.Broker)
	}

	return simpleAuthMiddleware(b.baseURL)
}

func (b *Broker) GetDashboardURL(groupID, clusterName string) string {
	return fmt.Sprintf("%s/v2/%s#clusters/detail/%s", b.baseURL, groupID, clusterName)
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
