package broker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/goccy/go-yaml"
	"github.com/gorilla/mux"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// Ensure broker adheres to the ServiceBroker interface.
var _ domain.ServiceBroker = new(Broker)

// Broker is responsible for translating OSB calls to Atlas API calls.
// Implements the domain.ServiceBroker interface making it easy to spin up
// an API server.
type Broker struct {
	logger       *zap.SugaredLogger
	whitelist    Whitelist
	credentials  *credentials.Credentials
	baseURL      string
	catalog      *catalog
	stateStorage StateStorage
}

// New creates a new Broker with a logger.
func New(logger *zap.SugaredLogger, credentials *credentials.Credentials, baseURL string, whitelist Whitelist, storage StateStorage) *Broker {
	b := &Broker{
		logger:       logger,
		credentials:  credentials,
		baseURL:      baseURL,
		whitelist:    whitelist,
		stateStorage: storage,
	}

	b.buildCatalog()
	return b
}

func (b *Broker) parsePlan(ctx dynamicplans.Context, planID string) (dp *dynamicplans.Plan, err error) {
	sp, ok := b.catalog.plans[planID]
	if !ok {
		err = fmt.Errorf("plan ID %q not found in catalog", planID)
		return
	}

	tpl, ok := sp.Metadata.AdditionalMetadata["template"].(dynamicplans.TemplateContainer)
	if !ok {
		err = errors.New("plan ID %q does not contain a valid plan template")
		return
	}

	raw := new(bytes.Buffer)
	err = tpl.Execute(raw, ctx.With("credentials", b.credentials))
	if err != nil {
		return
	}

	b.logger.Infow("Parsed plan", "plan", raw.String())

	dp = &dynamicplans.Plan{}
	if err = yaml.NewDecoder(raw).Decode(dp); err != nil {
		return
	}

	return dp, nil
}

func (b *Broker) getInstancePlan(ctx context.Context, instanceID string) (*dynamicplans.Plan, error) {
	i, err := b.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	params, ok := i.Parameters.(bson.D)
	if !ok {
		b.logger.Errorf("%#v", i)
		return nil, fmt.Errorf("instance metadata has the wrong type %T", i.Parameters)
	}

	p, found := params.Map()["plan"]
	if !found {
		return nil, fmt.Errorf("plan not found in instance metadata")
	}

	d, ok := p.(bson.D)
	if !ok {
		return nil, fmt.Errorf("instance metadata plan has the wrong type %T", p)
	}

	plan := dynamicplans.Plan{}
	bytes, err := bson.Marshal(d)
	if err != nil {
		return nil, err
	}

	err = bson.Unmarshal(bytes, &plan)
	return &plan, err
}

func (b *Broker) createClient(k credentials.APIKey) (*mongodbatlas.Client, error) {
	hc, err := digest.NewTransport(k.PublicKey, k.PrivateKey).Client()
	if err != nil {
		return nil, err
	}

	return mongodbatlas.New(hc, mongodbatlas.SetBaseURL(b.baseURL))
}

func (b *Broker) getPlan(ctx context.Context, instanceID string, planID string, planCtx dynamicplans.Context) (dp *dynamicplans.Plan, err error) {
	// existing instance: try to get from state store
	dp, err = b.getInstancePlan(ctx, instanceID)
	if err == nil {
		return
	}

	// new instance: get from plan
	dp, err = b.parsePlan(planCtx, planID)
	if err != nil {
		return
	}

	if dp.Project == nil {
		err = fmt.Errorf("missing Project in plan definition")
		return
	}

	return
}

func (b *Broker) getClient(ctx context.Context, instanceID string, planID string, planCtx dynamicplans.Context) (client *mongodbatlas.Client, dp *dynamicplans.Plan, err error) {
	dp, err = b.getPlan(ctx, instanceID, planID, planCtx)
	if err != nil {
		return
	}

	if dp.Project.ID != "" {
		c, ok := b.credentials.Projects[dp.Project.ID]
		if !ok {
			err = fmt.Errorf("credentials for project ID %q not found", dp.Project.ID)
			return
		}

		client, err = b.createClient(c)
		return
	}

	if dp.Project.OrgID != "" {
		oid := dp.Project.OrgID
		c, ok := b.credentials.Orgs[oid]
		if !ok {
			err = fmt.Errorf("credentials for org ID %q not found", oid)
			return
		}

		client, err = b.createClient(c)
		if err != nil {
			return
		}

		// try to merge existing project into plan, don't error out if not found
		var existing *mongodbatlas.Project
		existing, _, err = client.Projects.GetOneProjectByName(ctx, dp.Project.Name)
		if err == nil {
			dp.Project = existing
			return
		} else {
			err = nil
		}
		return
	}

	err = fmt.Errorf("project info must contain either ID or OrgID & project name, got %+v", dp.Project)
	return
}

func (b *Broker) AuthMiddleware() mux.MiddlewareFunc {
	if b.credentials != nil {
		return authMiddleware(*b.credentials.Broker)
	}

	return simpleAuthMiddleware(b.baseURL)
}

func (b *Broker) GetDashboardURL(groupID, clusterName string) string {
	apiUrl, err := url.Parse(b.baseURL)
	if err != nil {
		return err.Error()
	}
	apiUrl.Path = fmt.Sprintf("/v2/%s", groupID)
	return apiUrl.String() + fmt.Sprintf("#clusters/detail/%s", clusterName)
}

// TODO: do something about this!
// atlasToAPIError converts an Atlas error to a OSB response error.
func atlasToAPIError(err error) error {
	// switch err {
	// case atlas.ErrClusterNotFound:
	// 	return apiresponses.ErrInstanceDoesNotExist
	// case atlas.ErrClusterAlreadyExists:
	// 	return apiresponses.ErrInstanceAlreadyExists
	// case atlas.ErrUserAlreadyExists:
	// 	return apiresponses.ErrBindingAlreadyExists
	// case atlas.ErrUserNotFound:
	// 	return apiresponses.ErrBindingDoesNotExist
	// case atlas.ErrUnauthorized:
	// 	return apiresponses.NewFailureResponse(err, http.StatusUnauthorized, "")
	// }

	// Fall back on returning the error again if no others match.
	// Will result in a 500 Internal Server Error.
	return err
}
