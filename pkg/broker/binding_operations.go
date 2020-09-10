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
	"net/http"
	"net/url"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pkg/errors"
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
	logger := b.funcLogger()
	logger.Infow("Creating binding", "instance_id", instanceID, "binding_id", bindingID, "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
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

	// Fetch the cluster from Atlas to ensure it exists.
	cluster, _, err := client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		return
	}

	// Generate a cryptographically secure random password.
	password, err := generatePassword()
	if err != nil {
		logger.Errorw("Failed to generate password", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		err = errors.New("failed to generate binding password")
		return
	}

	user, err := b.userFromParams(bindingID, password, details.RawParameters, p)
	if err != nil {
		logger.Errorw("Couldn't create user from the passed parameters", "error", err, "instance_id", instanceID, "binding_id", bindingID, "details", details)
		return
	}

	// Create a new Atlas database user from the generated definition.
	_, _, err = client.DatabaseUsers.Create(ctx, p.Project.ID, user)
	if err != nil {
		logger.Errorw("Failed to create Atlas database user", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		return
	}

	logger.Infow("Successfully created Atlas database user", "instance_id", instanceID, "binding_id", bindingID)

	cs, err := url.Parse(cluster.ConnectionStrings.StandardSrv)
	if err != nil {
		logger.Errorw("Failed to parse connection string", "error", err, "connString", cluster.ConnectionStrings.StandardSrv)
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

	ss, err := b.stateStorage(p.Project.OrgID)
	if err != nil {
		return spec, err
	}

	_, err = ss.Put(ctx, bindingID, spec)
	return
}

// Unbind will delete the database user for a specific binding. The database
// user should have the binding ID as its username.
func (b Broker) Unbind(ctx context.Context, instanceID string, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (spec domain.UnbindSpec, err error) {
	logger := b.funcLogger()
	logger.Infow("Releasing binding", "instance_id", instanceID, "binding_id", bindingID, "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		return
	}

	// Fetch the cluster from Atlas to ensure it exists.
	_, _, err = client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		return
	}

	// Delete database user which has the binding ID as its username.
	_, err = client.DatabaseUsers.Delete(ctx, "admin", p.Project.ID, bindingID)
	if err != nil {
		logger.Errorw("Failed to delete Atlas database user", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		return
	}

	logger.Infow("Successfully deleted Atlas database user", "instance_id", instanceID, "binding_id", bindingID)

	spec = domain.UnbindSpec{}
	ss, err := b.stateStorage(p.Project.OrgID)
	if err != nil {
		return
	}

	err = ss.DeleteOne(ctx, bindingID)
	return
}

// GetBinding is currently not supported as specified by the
// BindingsRetrievable setting in the service catalog.
func (b Broker) GetBinding(ctx context.Context, instanceID string, bindingID string) (spec domain.GetBindingSpec, err error) {
	logger := b.funcLogger()
	logger.Infow("Retrieving binding", "instance_id", instanceID, "binding_id", bindingID)
	err = b.getState(ctx, bindingID, &spec)
	return
}

// LastBindingOperation should fetch the status of the last creation/deletion
// of a database user.
func (b Broker) LastBindingOperation(ctx context.Context, instanceID string, bindingID string, details domain.PollDetails) (resp domain.LastOperation, err error) {
	logger := b.funcLogger()
	logger.Infow("Fetching state of last binding operation", "instance_id", instanceID, "details", details)

	resp.State = domain.Failed

	// brokerapi will NOT update service state if we return any error, so... we won't?
	defer func() {
		if err != nil {
			resp.State = domain.Failed
			resp.Description = err.Error()
			err = nil
		}
	}()

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		return
	}

	_, r, err := client.DatabaseUsers.Get(ctx, "admin", p.Project.ID, bindingID)
	switch r.StatusCode {
	case http.StatusNotFound:
	default:
		err = errors.Wrap(err, "cannot get binding")
		logger.Errorw("Failed to get binding", "error", err, "instance_id", instanceID, "binding_id", bindingID)
		return
	}

	return resp, err
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

func (b *Broker) userFromParams(bindingID string, password string, rawParams []byte, plan *dynamicplans.Plan) (*mongodbatlas.DatabaseUser, error) {
	logger := b.funcLogger()
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
		logger.Warn(`No "databaseName" in User, setting to "admin" for Atlas.`)
		params.User.DatabaseName = "admin"
	}

	if plan.Settings != nil {
		if overrideDBName, ok := plan.Settings[overrideBindDB]; ok {
			overrideDBRole, ok := plan.Settings[overrideBindDBRole]
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
