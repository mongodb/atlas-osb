package broker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"encoding/json"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/goccy/go-yaml"
	"github.com/gorilla/mux"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	client      *mongo.Client
}

// New creates a new Broker with a logger.
func New(logger *zap.SugaredLogger, credentials *credentials.Credentials, baseURL string, whitelist Whitelist, client *mongo.Client, mode Mode) *Broker {
	b := &Broker{
		logger:      logger,
		credentials: credentials,
		baseURL:     baseURL,
		whitelist:   whitelist,
		client:      client,
		mode:        mode,
	}

	if err := b.buildCatalog(); err != nil {
		logger.Fatalw("Cannot build service catalog", "error", err)
	}

	return b
}

func (b *Broker) parsePlan(ctx dynamicplans.Context, planID string) (dp dynamicplans.Plan, err error) {
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

	if err = yaml.NewDecoder(raw).Decode(&dp); err != nil {
		return
	}

    // Attempt to merge in any other values as plan instance data
    pb, _ := json.Marshal(ctx)
    err = json.Unmarshal(pb, &dp)
    if err != nil {
        b.logger.Errorw("Error trying to merge in planContext as plan instance","err",err)
    } else {
        b.logger.Infow("Merged final cluster:",  "dp.Cluster", dp.Cluster)
    }

	return dp, nil
}

func (b *Broker) getInstanceState(ctx context.Context, instanceID string) (primitive.M, error) {
	i, err := b.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	p, ok := i.Parameters.(primitive.D)
	if !ok {
		return nil, fmt.Errorf("instance metadata has the wrong type %T", i.Parameters)
	}

	return p.Map(), nil
}

func (b *Broker) getGroupIDByInstanceID(ctx context.Context, instanceID string) (string, error) {
	s, err := b.getInstanceState(ctx, instanceID)
	if err != nil {
		// no metadata - not an error in our case
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}

	gidi, ok := s["groupID"]
	if !ok {
		return "", fmt.Errorf("groupID not found in instance metadata for %q", instanceID)
	}

	gid, ok := gidi.(string)
	if !ok {
		return "", fmt.Errorf("groupID from instance metadata has the wrong type %T", gidi)
	}

	return gid, nil
}

func (b *Broker) getClusterNameByInstanceID(ctx context.Context, instanceID string) (string, error) {
	if b.client == nil {
		return NormalizeClusterName(instanceID), nil
	}

	s, err := b.getInstanceState(ctx, instanceID)
	if err != nil {
		return "", err
	}

	ci, ok := s["clusterName"]
	if !ok {
		return "", fmt.Errorf("clusterName not found in instance metadata for %q", instanceID)
	}

	c, ok := ci.(string)
	if !ok {
		return "", fmt.Errorf("clusterName from instance metadata has the wrong type %T", ci)
	}

	return c, nil
}

func (b *Broker) getClient(ctx context.Context, instanceID string, planID string, planCtx dynamicplans.Context) (client *mongodbatlas.Client, gid string, err error) {
	switch b.mode {
	case BasicAuth:
		client, err = atlasClientFromContext(ctx)
		if err != nil {
			return
		}
		gid, err = groupIDFromContext(ctx)
		return client, gid, err

	case MultiGroup:
		panic("not implemented")

	case MultiGroupAutoPlans:
		gid, err = b.catalog.findGroupIDByPlanID(planID)
		if err != nil {
			return nil, gid, err
		}

	case DynamicPlans:
		// try to get groupID for existing instances
		gid, err = b.getGroupIDByInstanceID(ctx, instanceID)
		if err != nil {
			return
		}

		if gid != "" {
			break
		}

		// new instance: get groupID from params
		dp := dynamicplans.Plan{}
		dp, err = b.parsePlan(planCtx, planID)
		if err != nil {
			return
		}

		if dp.Project == nil {
			err = fmt.Errorf("missing Project in plan definition")
			return
		}

		// use existing project
		if dp.Project.ID != "" {
			gid = dp.Project.ID
			break
		}

		if dp.Project.OrgID != "" {
			oid := dp.Project.OrgID
			c, ok := b.credentials.Orgs[oid]
			if !ok {
                keys := make([]string, len(b.credentials.Orgs))
                i := 0
                for k := range b.credentials.Orgs {
                    keys[i] = k
                    i++
                }
                c, ok = b.credentials.Orgs[keys[0]]
			    if !ok {
				    return nil, "", fmt.Errorf("credentials for org ID %q not found", oid)
                } 
                // TODO -- log that we just grab the 1st org key there
			}

			hc, err := digest.NewTransport(c.PublicKey, c.PrivateKey).Client()
			if err != nil {
				return nil, "", err
			}

			client, err = mongodbatlas.New(hc, mongodbatlas.SetBaseURL(b.baseURL))

			if dp.Project.Name != "" {
				p, _, err := client.Projects.GetOneProjectByName(ctx, dp.Project.Name)
				if err == nil {
					return client, p.ID, err
				}
			}
			return client, "", err
		}

	default:
		panic("invalid broker mode")
	}

	c, ok := b.credentials.Projects[gid]
	if !ok {
		return nil, gid, fmt.Errorf("credentials for project ID %q not found", gid)
	}

	hc, err := digest.NewTransport(c.PublicKey, c.PrivateKey).Client()
	if err != nil {
		return nil, gid, err
	}

	client, err = mongodbatlas.New(hc, mongodbatlas.SetBaseURL(b.baseURL))
	return client, gid, err
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
