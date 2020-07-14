package broker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
)

// ConnectionDetails will be returned when a new binding is created.
type ConnectionDetails struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	URI              string `json:"uri"`
	ConnectionString string `json:"connectionString"`
}

// Bind will create a new database user with a username matching the binding ID
// and a randomly generated password. The user credentials will be returned back.
func (b Broker) Bind(ctx context.Context, instanceID string, bindingID string, details domain.BindDetails, asyncAllowed bool) (spec domain.Binding, err error) {
	b.logger.Infow("Creating binding", "instance_id", instanceID, "binding_id", bindingID, "details", details)

	planContext := dynamicplans.Context{
		"instance_id": instanceID,
	}
	if len(details.RawParameters) > 0 {
		err = json.Unmarshal(details.RawParameters, &planContext)
		if err != nil {
			return
		}
	}

	if len(details.RawContext) > 0 {
		err = json.Unmarshal(details.RawContext, &planContext)
		if err != nil {
			return
		}
	}

	client, gid, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	// The service_id and plan_id are required to be valid per the specification, despite
	// not being used for bindings. We look them up to ensure they can be found in the catalog.
	_, ok := b.catalog.providers[details.ServiceID]
	if !ok {
		return spec, fmt.Errorf("service ID %q not found in catalog", details.ServiceID)
	}

	_, ok = b.catalog.plans[details.PlanID]
	if !ok {
		return spec, fmt.Errorf("plan ID %q not found in catalog", details.PlanID)
	}

	name, err := b.getClusterNameByInstanceID(ctx, instanceID)
	if err != nil {
		return
	}

	// Fetch the cluster from Atlas to ensure it exists.
	cluster, _, err := client.Clusters.Get(ctx, gid, name)
	if err != nil {
		b.logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		err = atlasToAPIError(err)
		return
	}

	// Generate a cryptographically secure random password.
	password, err := generatePassword()
	if err != nil {
		b.logger.Errorw("Failed to generate password", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		err = errors.New("Failed to generate binding password")
		return
	}

	// Construct a cluster definition from the instance ID, service, plan, and params.
	user, err := userFromParams(bindingID, password, details.RawParameters,&b)
	if err != nil {
		b.logger.Errorw("Couldn't create user from the passed parameters", "error", err, "instance_id", instanceID, "binding_id", bindingID, "details", details)
		return
	}

	// Create a new Atlas database user from the generated definition.
	_, _, err = client.DatabaseUsers.Create(ctx, gid, user)
	if err != nil {
		b.logger.Errorw("Failed to create Atlas database user", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully created Atlas database user", "instance_id", instanceID, "binding_id", bindingID)

	cs, err := url.Parse(cluster.ConnectionStrings.StandardSrv)
	if err != nil {
		b.logger.Errorw("Failed to parse connection string", "error", err, "connString", cluster.ConnectionStrings.StandardSrv)
	}

	b.logger.Infow("New User ConnectionString", "connectionString", cs)

	cs.User = url.UserPassword(user.Username, user.Password)
	cs.Path = user.DatabaseName

	spec = domain.Binding{
		Credentials: ConnectionDetails{
			Username:         bindingID,
			Password:         password,
			URI:              cluster.SrvAddress,
			ConnectionString: cs.String(),
		},
	}
	return
}

// Unbind will delete the database user for a specific binding. The database
// user should have the binding ID as its username.
func (b Broker) Unbind(ctx context.Context, instanceID string, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (spec domain.UnbindSpec, err error) {
	b.logger.Infow("Releasing binding", "instance_id", instanceID, "binding_id", bindingID, "details", details)

	planContext := dynamicplans.Context{
		"instance_id": instanceID,
	}
	client, gid, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	name, err := b.getClusterNameByInstanceID(ctx, instanceID)
	if err != nil {
		return
	}

	// Fetch the cluster from Atlas to ensure it exists.
	_, _, err = client.Clusters.Get(ctx, gid, name)
	if err != nil {
		b.logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		err = atlasToAPIError(err)
		return
	}

	// Delete database user which has the binding ID as its username.
	_, err = client.DatabaseUsers.Delete(ctx, "admin", gid, bindingID)
	if err != nil {
		b.logger.Errorw("Failed to delete Atlas database user", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully deleted Atlas database user", "instance_id", instanceID, "binding_id", bindingID)

	spec = domain.UnbindSpec{}
	return
}

// GetBinding is currently not supported as specified by the
// BindingsRetrievable setting in the service catalog.
func (b Broker) GetBinding(ctx context.Context, instanceID string, bindingID string) (spec domain.GetBindingSpec, err error) {
	b.logger.Infow("Retrieving binding", "instance_id", instanceID, "binding_id", bindingID)

	err = apiresponses.NewFailureResponse(fmt.Errorf("Unknown binding ID %s", bindingID), 404, "get-binding")
	return
}

// LastBindingOperation should fetch the status of the last creation/deletion
// of a database user.
func (b Broker) LastBindingOperation(ctx context.Context, instanceID string, bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	panic("not implemented")
}

// generatePassword will generate a cryptographically secure password.
// The password will be base64 encoded for easy usage.
func generatePassword() (string, error) {
	const numberOfBytes = 32
	b := make([]byte, numberOfBytes)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

func userFromParams(bindingID string, password string, rawParams []byte, broker *Broker) (*mongodbatlas.DatabaseUser, error) {
	// Set up a params object which will be used for deserialiation.
	params := struct {
		User *mongodbatlas.DatabaseUser `json:"user"`
	}{
		&mongodbatlas.DatabaseUser{},
	}

	// If params were passed we unmarshal them into the params object.
	if len(rawParams) > 0 {
		err := json.Unmarshal(rawParams, &params)
		if err != nil {
			return nil, err
		}
	}

	// Set binding ID as username and add password.
	params.User.Username = bindingID
	params.User.Password = password
    if len(params.User.DatabaseName) == 0 {
        params.User.DatabaseName = "admin"
    }
    broker.logger.Infow("userFromParams",params,"params")
	// If no role is specified we default to read/write on any database.
	// This is the default role when creating a user through the Atlas UI.
	if len(params.User.Roles) == 0 {
		params.User.Roles = []mongodbatlas.Role{
			{
				RoleName:     "readWriteAnyDatabase",
				DatabaseName: "admin",
			},
		}
	}

	return params.User, nil
}
