// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
	"go.mongodb.org/atlas/mongodbatlas"
)

const (
	overrideBindDB     = "overrideBindDB"
	overrideBindDBRole = "overrideBindDBRole"
)

// ConnectionDetails will be returned when a new binding is created.
type ConnectionDetails struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	URI              string `json:"uri"`
	ConnectionString string `json:"connectionString"`
	Database         string `json:"database"`
}

// Bind will create a new database user with a username matching the binding ID
// and a randomly generated password. The user credentials will be returned back.
func (b Broker) Bind(ctx context.Context, instanceID string, bindingID string, details domain.BindDetails, asyncAllowed bool) (spec domain.Binding, err error) {
	logger := b.funcLogger().With("instance_id", instanceID, "binding_id", bindingID)
	logger.Infow("Creating binding", "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		logger.Errorw("Failed to get existing client", "error", err)

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

	// Fetch the cluster from Atlas to ensure it exists.
	cluster, _, err := client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to get existing cluster", "error", err)

		return
	}

	// Generate a cryptographically secure random password.
	password, err := generatePassword()
	if err != nil {
		logger.Errorw("Failed to generate password", "error", err)
		err = errors.New("failed to generate binding password")

		return
	}

	user, err := b.userFromParams(bindingID, password, details.RawParameters, p)
	if err != nil {
		logger.Errorw("Couldn't create user from the passed parameters", "error", err, "details", details)

		return
	}

	// Create a new Atlas database user from the generated definition.
	_, _, err = client.DatabaseUsers.Create(ctx, p.Project.ID, user)
	if err != nil {
		logger.Errorw("Failed to create Atlas database user", "error", err)

		return
	}

	logger.Infow("Successfully created Atlas database user")

	cs, err := url.Parse(cluster.ConnectionStrings.StandardSrv)
	if len(cluster.ConnectionStrings.PrivateEndpoint) != 0 {
		logger.Infow("Using private connection string")
		for _, e := range cluster.ConnectionStrings.PrivateEndpoint {
			cs, err = url.Parse(e.SRVConnectionString)
			if err != nil {
				logger.Errorw("Failed to parse private connection string", "error", err)

				continue
			}

			break
		}
	}
	if err != nil {
		logger.Errorw("Failed to parse connection strings", "error", err,
			"standard", cluster.ConnectionStrings.StandardSrv)

		return
	}

	cs.Path = user.DatabaseName

	connDetails := ConnectionDetails{
		Username: bindingID,
		Password: password,
	}

	if len(user.Roles) > 0 {
		cs.Path = user.Roles[0].DatabaseName
		logger.Infow("Detected roles, override the name of the db to connect", "connectionString", cs)
	}

	logger.Infow("New User ConnectionString", "connectionString", cs)

	cs.User = url.UserPassword(user.Username, user.Password)
	connDetails.ConnectionString = cs.String()
	connDetails.Database = cs.Path
	connDetails.URI = cs.String()

	spec = domain.Binding{
		Credentials: connDetails,
	}

	return
}

// Unbind will delete the database user for a specific binding. The database
// user should have the binding ID as its username.
func (b Broker) Unbind(ctx context.Context, instanceID string, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (spec domain.UnbindSpec, err error) {
	logger := b.funcLogger().With("instance_id", instanceID, "binding_id", bindingID)
	logger.Infow("Releasing binding", "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		logger.Errorw("Failed to get existing client", "error", err)

		return
	}

	// Fetch the cluster from Atlas to ensure it exists.
	_, _, err = client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to get existing cluster", "error", err)

		return
	}

	// Delete database user which has the binding ID as its username.
	_, err = client.DatabaseUsers.Delete(ctx, "admin", p.Project.ID, bindingID)
	if err != nil {
		logger.Errorw("Failed to delete Atlas database user", "error", err)

		return
	}

	logger.Infow("Successfully deleted Atlas database user")

	spec = domain.UnbindSpec{}

	return
}

// GetBinding is currently not supported as specified by the
// BindingsRetrievable setting in the service catalog.
func (b Broker) GetBinding(ctx context.Context, instanceID string, bindingID string) (spec domain.GetBindingSpec, err error) {
	logger := b.funcLogger().With("instance_id", instanceID, "binding_id", bindingID)
	logger.Infow("Retrieving binding")

	err = apiresponses.NewFailureResponse(fmt.Errorf("unknown binding ID %s", bindingID), 404, "get-binding")

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
		return "", errors.Wrap(err, "cannot read random bytes")
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

func (b *Broker) userFromParams(bindingID string, password string, rawParams []byte, plan *dynamicplans.Plan) (*mongodbatlas.DatabaseUser, error) {
	logger := b.funcLogger().With("binding_id", bindingID)
	// Set up a params object which will be used for deserialization.
	params := struct {
		User *mongodbatlas.DatabaseUser `json:"user"`
	}{
		&mongodbatlas.DatabaseUser{},
	}

	// If params were passed we unmarshal them into the params object.
	if len(rawParams) > 0 {
		err := json.Unmarshal(rawParams, &params)
		if err != nil {
			return nil, errors.Wrap(err, "cannot unmarshal raw parameters")
		}
	}

	// Set binding ID as username and add password.
	params.User.Username = bindingID
	params.User.Password = password
	if len(params.User.DatabaseName) == 0 {
		logger.Warn(`No "databaseName" in User, setting to "admin" for Atlas.`)
		params.User.DatabaseName = "admin"
	}

	if plan.Settings != nil {
		if overrideDBName, ok := plan.Settings[overrideBindDB].(string); ok {
			overrideDBRole, ok := plan.Settings[overrideBindDBRole].(string)
			if !ok {
				overrideDBRole = "readWrite"
			}
			overrideRole := mongodbatlas.Role{
				DatabaseName: overrideDBName,
				RoleName:     overrideDBRole,
			}
			logger.Warnw("DEPRECATED: Overriding bind DB settings", "overrideRole", overrideRole)
			params.User.Roles = append(params.User.Roles, overrideRole)
		}
	}

	if len(params.User.DatabaseName) == 0 {
		params.User.DatabaseName = "admin"
	}

	if len(params.User.Scopes) == 0 {
		params.User.Scopes = append(params.User.Scopes, mongodbatlas.Scope{
			Name: plan.Cluster.Name,
			Type: "CLUSTER",
		})
	}

	logger.Debugw("userFromParams", "params", params)

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
