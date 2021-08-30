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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/mongodb/atlas-osb/pkg/broker/privateendpoint"
	"github.com/mongodb/atlas-osb/pkg/broker/statestorage"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
	"go.mongodb.org/atlas/mongodbatlas"
)

// The different async operations that can be performed.
// These constants are returned during provisioning, deprovisioning, and
// updates and are subsequently included in async polls from the platform.
const (
	operationProvision   = "provision"
	operationDeprovision = "deprovision"
	operationUpdate      = "update"
)

// Provision will create a new Atlas cluster with the instance ID as its name.
// The process is always async.
func (b Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	logger := b.funcLogger().With("instance_id", instanceID)

	logger.Infow("Provisioning instance", "details", details)

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

	client, dp, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	if dp.Project.ID == "" {
		var newProject *mongodbatlas.Project
		newProject, _, err = client.Projects.Create(ctx, dp.Project)
		if err != nil {
			logger.Errorw("Cannot create project", "error", err, "project", dp.Project)

			return
		}

		dp.Project = newProject
		err = b.createOrUpdateResources(ctx, client, dp, dp)
		if err != nil {
			logger.Errorw("Cannot update resource", "error", err, "project", dp.Project)

			return
		}

		dp.Project.ID = newProject.ID
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired

		return
	}

	// Construct a cluster definition from the instance ID, service, plan, and params.
	logger.Infow("Creating cluster", "instance_name", planContext["instance_name"])
	// TODO - add this context info about k8s/namespace or pcf space into labels

	planEnc, err := encodePlan(*dp)
	if err != nil {
		return
	}

	s := domain.GetInstanceDetailsSpec{
		PlanID:       details.PlanID,
		ServiceID:    details.ServiceID,
		DashboardURL: b.GetDashboardURL(dp.Project.ID, dp.Cluster.Name),
		Parameters:   planEnc,
	}

	state, err := b.getState(ctx, dp.Project.OrgID)
	if err != nil {
		return
	}

	v, err := state.Put(ctx, instanceID, &s)
	if err != nil {
		logger.Errorw("Error during provision, broker maintenance:", "err", err)

		return
	}
	logger.Infow("Inserted new state value", "v", v)

	defer func() {
		if err != nil {
			_ = state.DeleteOne(ctx, instanceID)
		}
	}()

	// Create a new Atlas cluster from the generated definition
	resultingCluster, _, err := client.Clusters.Create(ctx, dp.Project.ID, dp.Cluster)
	if err != nil {
		logger.Errorw("Failed to create Atlas cluster", "error", err, "cluster", dp.Cluster)

		return
	}

	logger.Infow("Successfully started Atlas creation process", "cluster", resultingCluster)

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: operationProvision,
		DashboardURL:  b.GetDashboardURL(dp.Project.ID, resultingCluster.Name),
	}, nil
}

func (b *Broker) createOrUpdateResources(ctx context.Context, client *mongodbatlas.Client, newPlan *dynamicplans.Plan, oldPlan *dynamicplans.Plan) error {
	logger := b.funcLogger()

	for _, u := range newPlan.DatabaseUsers {
		if len(u.Scopes) == 0 {
			u.Scopes = append(u.Scopes, mongodbatlas.Scope{
				Name: newPlan.Cluster.Name,
				Type: "CLUSTER",
			})
		}

		_, r, err := client.DatabaseUsers.Create(ctx, oldPlan.Project.ID, u)
		if err != nil {
			if r.StatusCode != http.StatusConflict {
				return errors.Wrap(err, "cannot create Database User")
			}

			_, _, err = client.DatabaseUsers.Update(ctx, oldPlan.Project.ID, u.Username, u)
			if err != nil {
				return errors.Wrap(err, "cannot update Database User")
			}
		}
	}

	// keep support for the deprecated IPWhitelists
	if len(newPlan.IPWhitelists) > 0 { // nolint
		// note: Create() is identical to Update()
		_, _, err := client.ProjectIPWhitelist.Create(ctx, oldPlan.Project.ID, newPlan.IPWhitelists) // nolint
		if err != nil {
			return errors.Wrap(err, "cannot create/update IP Whitelist")
		}
	}

	if len(newPlan.IPAccessLists) > 0 {
		// note: Create() is identical to Update()
		_, _, err := client.ProjectIPAccessList.Create(ctx, oldPlan.Project.ID, newPlan.IPAccessLists)
		if err != nil {
			return errors.Wrap(err, "cannot create/update IP Access List")
		}
	}

	// create and populater the set with IPs from the plan
	planIPAccessListItems := make(map[string]struct{})
	for _, item := range newPlan.IPAccessLists {
		planIPAccessListItems[item.IPAddress] = struct{}{}
	}
	logger.Debugw("IP Access List from the plan", "IPs", planIPAccessListItems)

	atlasAccessLists, _, err := client.ProjectIPAccessList.List(ctx, oldPlan.Project.ID, nil)
	if err != nil {
		return errors.Wrap(err, "cannot get IP Access Lists from Atlas")
	}
	for _, item := range atlasAccessLists.Results {
		// delete all IPs which are not in the plan
		if _, ok := planIPAccessListItems[item.CIDRBlock]; !ok {
			logger.Debugw("Deleting IP Access List Item", "cidrBlock", item.CIDRBlock, "item", item)
			_, err := client.ProjectIPAccessList.Delete(ctx, oldPlan.Project.ID, item.CIDRBlock)
			if err != nil {
				logger.Errorw("Failed to delete an item from IP Access List", "err", err)
			}
		}
	}

	for _, i := range newPlan.Integrations {
		_, _, err := client.Integrations.Replace(ctx, oldPlan.Project.ID, i.Type, i)
		if err != nil {
			return errors.Wrap(err, "cannot create Third-Party Integration")
		}
	}

	if err := b.syncPrivateEndpoints(ctx, client, newPlan, oldPlan); err != nil {
		return errors.Wrap(err, "cannot sync Private Endpoints")
	}

	return nil
}

func (b *Broker) syncPrivateEndpoints(ctx context.Context, client *mongodbatlas.Client, newPlan *dynamicplans.Plan, oldPlan *dynamicplans.Plan) error {
	logger := b.funcLogger()

	peProvider := "AZURE" // this is hardcoded cause only one provider is supported for now

	atlasPrivateEndpoints, _, err := client.PrivateEndpoints.List(ctx, oldPlan.Project.ID, peProvider, nil)
	if err != nil {
		return errors.Wrap(err, "cannot get Private Endpoints from Atlas")
	}
	atlasPrivateEndpoints = b.populateConnections(atlasPrivateEndpoints)

	planPrivateEndpoints := make(map[string]struct{})
	for _, endpoint := range newPlan.PrivateEndpoints {
		if endpoint.ID != "" {
			// endpoint service already created
			planPrivateEndpoints[endpoint.ID] = struct{}{}

			continue
		}

		logger.Debugw("Atlas PEs", "endpoints", atlasPrivateEndpoints)

		if connID, exists := b.privateEndpointExists(endpoint.Provider, endpoint.EndpointName, atlasPrivateEndpoints); exists {
			planPrivateEndpoints[connID] = struct{}{}
			endpoint.ID = connID

			continue
		}

		logger.Debugw("start creation process for PrivateEndpoints", "project ID", oldPlan.Project.ID)
		conn, _, err := client.PrivateEndpoints.Create(ctx, oldPlan.Project.ID, &mongodbatlas.PrivateEndpointConnection{
			ProviderName: endpoint.Provider,
			Region:       endpoint.Region,
		})
		if err != nil {
			return errors.Wrap(err, "cannot create Private Endpoint Service")
		} else {
			endpoint.ID = conn.ID
			planPrivateEndpoints[endpoint.ID] = struct{}{}
		}
	}

	logger.Debugw("Private Endpoints from the plan", "PE names", planPrivateEndpoints)

	for _, peConnection := range atlasPrivateEndpoints {
		// delete all PE endpoints which are not in the plan
		if _, isPlanned := planPrivateEndpoints[peConnection.ID]; !isPlanned {
			logger.Debugw("Deleting Private Endpoint", "connection", peConnection)

			for _, endpoint := range oldPlan.PrivateEndpoints {
				if endpoint.EndpointName == peConnection.EndpointServiceName && endpoint.Provider == peConnection.ProviderName {
					if _, err := privateendpoint.Delete(ctx, endpoint); err != nil {
						logger.Errorw("Failed to delete Private Endpoint from Azure", "error", err, "endpoint", endpoint.EndpointName)
					}
				}
			}

			for _, peID := range peConnection.PrivateEndpoints {
				if _, err := client.PrivateEndpoints.DeleteOnePrivateEndpoint(ctx, oldPlan.Project.ID, peProvider, peConnection.ID, peID); err != nil {
					logger.Errorw("Failed to delete Private Endpoint from Atlas", "error", err, "pe", peID)
				}
			}

			if _, err := client.PrivateEndpoints.Delete(ctx, oldPlan.Project.ID, peProvider, peConnection.ID); err != nil {
				logger.Errorw("Failed to delete Private Endpoint Service from Atlas", "error", err, "pe", peConnection)
			}
		}
	}

	return nil
}

func (b *Broker) privateEndpointExists(provider string, name string, connections []mongodbatlas.PrivateEndpointConnection) (string, bool) {
	logger := b.funcLogger()

	for _, peConnection := range connections {
		logger.Infow("privateEndpointExists func", "conn", peConnection)
		if peConnection.ProviderName == provider && peConnection.EndpointServiceName == name {
			return peConnection.ID, true
		}
	}

	return "", false
}

func (b *Broker) populateConnections(connections []mongodbatlas.PrivateEndpointConnection) []mongodbatlas.PrivateEndpointConnection {
	logger := b.funcLogger()
	r := regexp.MustCompile("/([^/]+)/resourceGroups/([^/]+)/providers/([^/]+)/privateEndpoints/([^/]+)$")
	for connIdx := range connections {
		for _, pe := range connections[connIdx].PrivateEndpoints {
			res := r.FindAllStringSubmatch(pe, -1)
			if len(res) == 1 && len(res[0]) == 5 {
				m1 := res[0]
				sub, group, prov, name := m1[1], m1[2], m1[3], m1[4]
				logger.Debugw("populateConnections", "sub", sub, "group", group, "prov", prov, "name", name)
				connections[connIdx].ProviderName = prov
				connections[connIdx].EndpointServiceName = name

				if connections[connIdx].ProviderName == "Microsoft.Network" {
					connections[connIdx].ProviderName = "AZURE"
				}
			}
		}
	}

	return connections
}

// TODO: this retry logic is clunky, come up with something better?
func (b *Broker) postCreateResources(ctx context.Context, client *mongodbatlas.Client, dp *dynamicplans.Plan) (retry bool, err error) {
	logger := b.funcLogger()

	logger.Debugw("Setup PrivateEndpoints", "PrivateEndpoints", dp.PrivateEndpoints)
	for _, endpoint := range dp.PrivateEndpoints {
		if endpoint.ID == "" {
			continue
		}

		atlasService, _, err := client.PrivateEndpoints.Get(ctx, dp.Project.ID, endpoint.Provider, endpoint.ID)
		if err != nil {
			return false, errors.Wrap(err, "cannot get Private Endpoint Service")
		}

		switch atlasService.Status {
		case "INITIATING", "WAITING_FOR_USER":
			retry = true

			continue

		case "AVAILABLE":
			break

		default:
			return false, errors.Wrapf(err, "Private Endpoint service is in the wrong state: %s", atlasService.Status)
		}

		logger.Debugw("Creating private endpoint", "endpoint", endpoint.ID)
		future, err := privateendpoint.Create(ctx, endpoint, atlasService)
		if err != nil {
			return false, errors.Wrap(err, "cannot create Private Endpoint in Azure")
		}

		pe, err := future()
		if err != nil {
			logger.Debugw("PrivateEndpoint not ready; retry", "PrivateEndpoint", pe)
			retry = true

			continue
		}

		if pe.ID == nil {
			continue
		}

		addr, err := privateendpoint.GetIPAddress(ctx, pe, endpoint)
		if err != nil {
			return false, errors.Wrap(err, "cannot get IP address for Private Endpoint")
		}

		existing, r, err := client.PrivateEndpoints.GetOnePrivateEndpoint(ctx, dp.Project.ID, endpoint.Provider, endpoint.ID, *pe.ID)
		if err != nil {
			if r == nil || r.StatusCode != http.StatusNotFound {
				return false, errors.Wrap(err, "cannot get Private Endpoint")
			}
		}

		if existing == nil {
			_, _, err = client.PrivateEndpoints.AddOnePrivateEndpoint(ctx, dp.Project.ID, endpoint.Provider, endpoint.ID, &mongodbatlas.InterfaceEndpointConnection{
				ID:                       *pe.ID,
				PrivateEndpointIPAddress: addr,
			})
			if err != nil {
				return false, errors.Wrap(err, "cannot add Private Endpoint to Endpoint Service")
			}

			continue
		}

		if existing.PrivateEndpointIPAddress == addr {
			continue
		}

		_, err = client.PrivateEndpoints.DeleteOnePrivateEndpoint(ctx, dp.Project.ID, endpoint.Provider, endpoint.ID, *pe.ID)
		if err != nil {
			return false, errors.Wrap(err, "cannot delete Private Endpoint")
		}

		retry = true
	}

	return
}

// Update will change the configuration of an existing Atlas cluster asynchronously.
func (b Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (spec domain.UpdateServiceSpec, err error) {
	logger := b.funcLogger().With("instance_id", instanceID)
	logger.Infow("Updating instance", "details", details)

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

	logger.Infow("Update() planContext merged with details.parameters&context", "planContext", planContext)
	client, oldPlan, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired

		return
	}

	// special case: pause/unpause
	if paused, ok := planContext["paused"].(bool); ok {
		logger.Info("Special case: pause/unpause")
		request := &mongodbatlas.Cluster{
			Paused: &paused,
		}

		_, _, err = client.Clusters.Update(ctx, oldPlan.Project.ID, oldPlan.Cluster.Name, request)

		return domain.UpdateServiceSpec{
			IsAsync:       true,
			OperationData: operationUpdate,
			DashboardURL:  b.GetDashboardURL(oldPlan.Project.ID, oldPlan.Cluster.Name),
		}, errors.Wrap(err, "cannot update Cluster")
	}

	// special case: perform update operations
	if op, ok := planContext["op"].(string); ok {
		logger.Info("Special case: perform update operations")
		err = b.performOperation(ctx, client, planContext, oldPlan, op)

		return domain.UpdateServiceSpec{
			IsAsync:       false, // feature spec: "[the broker] will not maintain ANY state of the list of users in an Atlas project", so no "in progress" indicator
			OperationData: operationUpdate,
			DashboardURL:  b.GetDashboardURL(oldPlan.Project.ID, oldPlan.Cluster.Name),
		}, err
	}

	newPlan, err := b.parsePlan(planContext, details.PlanID)
	if err != nil {
		return
	}

	err = b.createOrUpdateResources(ctx, client, newPlan, oldPlan)
	if err != nil {
		logger.Errorw("Cannot update resources", "error", err)

		return
	}

	// Fetch the cluster from Atlas. The Atlas API requires an instance size to
	// be passed during updates (if there are other update to the provider, such
	// as region). The plan is not included in the OSB call unless it has changed
	// hence we need to fetch the current value from Atlas.
	existingCluster, _, err := client.Clusters.Get(ctx, oldPlan.Project.ID, oldPlan.Cluster.Name)
	if err != nil {
		return
	}

	// Atlas doesn't allow for cluster renaming - ignore any changes
	newPlan.Cluster.Name = existingCluster.Name

	if newPlan.Cluster.ProviderSettings.ProviderName == "AZURE" && len(newPlan.Cluster.ReplicationSpecs) > 0 {
		if !reflect.DeepEqual(oldPlan.Cluster.ReplicationSpecs, newPlan.Cluster.ReplicationSpecs) {
			err = errors.New("update operation for cluster.ReplicationSpecs is not supported by Atlas for Azure clusters; please remove this section from the plan or recreate the cluster")

			return
		}

		newPlan.Cluster.ReplicationSpecs = nil
	}

	resultingCluster, _, err := client.Clusters.Update(ctx, oldPlan.Project.ID, existingCluster.Name, newPlan.Cluster)
	if err != nil {
		logger.Errorw("Failed to update Atlas cluster", "error", err, "new_cluster", newPlan.Cluster)

		return
	}

	// update fields that can be safely updated
	oldPlan.Description = newPlan.Description
	oldPlan.Free = newPlan.Free
	oldPlan.Version = newPlan.Version
	oldPlan.Settings = newPlan.Settings
	oldPlan.Cluster = resultingCluster
	oldPlan.IPAccessLists = newPlan.IPAccessLists
	oldPlan.PrivateEndpoints = newPlan.PrivateEndpoints

	planEnc, err := encodePlan(*oldPlan)
	if err != nil {
		return
	}

	s := domain.GetInstanceDetailsSpec{
		PlanID:       details.PlanID,
		ServiceID:    details.ServiceID,
		DashboardURL: b.GetDashboardURL(oldPlan.Project.ID, oldPlan.Cluster.Name),
		Parameters:   planEnc,
	}

	state, err := b.getState(ctx, oldPlan.Project.OrgID)
	if err != nil {
		return
	}

	// TODO: make this error-out reversible?
	err = state.DeleteOne(ctx, instanceID)
	if err != nil {
		logger.Errorw("Error delete from state", "err", err)

		return
	}

	obj, err := state.Put(ctx, instanceID, &s)
	if err != nil {
		logger.Errorw("Error insert one from state", "err", err, "s", s)

		return
	}

	logger.Infow("Inserted into state", "obj", obj)
	logger.Infow("Successfully started Atlas cluster update process", "cluster", resultingCluster)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: operationUpdate,
		DashboardURL:  b.GetDashboardURL(oldPlan.Project.ID, resultingCluster.Name),
	}, nil
}

// Deprovision will destroy an Atlas cluster asynchronously.
func (b Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (spec domain.DeprovisionServiceSpec, err error) {
	logger := b.funcLogger().With("instance_id", instanceID)
	logger.Infow("Deprovisioning instance", "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired

		return
	}

	for _, peService := range p.PrivateEndpoints {
		if _, err := privateendpoint.Delete(ctx, peService); err != nil {
			logger.Errorw("Failed to delete Private Endpoint from Azure", "error", err, "peService", peService)
		}

		if peService.ID == "" {
			continue
		}

		conn, _, err := client.PrivateEndpoints.Get(ctx, p.Project.ID, peService.Provider, peService.ID)
		if err != nil {
			logger.Errorw("Failed to fetch Private Endpoint Service Connection from Atlas", "error", err, "peService", peService)
		}

		for _, peID := range conn.PrivateEndpoints {
			if _, err := client.PrivateEndpoints.DeleteOnePrivateEndpoint(ctx, p.Project.ID, peService.Provider, peService.ID, peID); err != nil {
				logger.Errorw("Failed to delete Private Endpoint from Atlas", "error", err, "pe", peID)
			}
		}
	}

	_, err = client.Clusters.Delete(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to delete Atlas cluster", "error", err)
	}

	for _, u := range p.DatabaseUsers {
		_, err = client.DatabaseUsers.Delete(ctx, u.DatabaseName, p.Project.ID, u.Username)
		if err != nil {
			logger.Errorw("failed to delete Database user", "error", err, "username", u.Username)
		}
	}

	logger.Infow("Successfully started Atlas Cluster & Project deletion process")

	return domain.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: operationDeprovision,
	}, nil
}

// GetInstance should fetch the stored instance from state storage
func (b Broker) GetInstance(ctx context.Context, instanceID string) (spec domain.GetInstanceDetailsSpec, err error) {
	logger := b.funcLogger().With("instanceID", instanceID)
	logger.Info("Fetching instance")

	spec, err = b.getInstance(ctx, instanceID)
	if err != nil {
		logger.Errorw("Unable to fetch instance", "err", err)

		return spec, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "get-instance")
	}

	if enc, ok := spec.Parameters.(string); ok {
		p, err := decodePlan(enc)
		if err != nil {
			return spec, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "get-instance")
		}
		spec.Parameters = p.SafeCopy()
	}

	return spec, nil
}

func (b Broker) getInstance(ctx context.Context, instanceID string) (spec domain.GetInstanceDetailsSpec, err error) {
	logger := b.funcLogger().With("instanceID", instanceID)

	for k, v := range b.credentials.Keys() {
		logger = logger.With("orgID", k)

		state, err := statestorage.Get(ctx, v, b.userAgent, b.cfg.AtlasURL, b.cfg.RealmURL, b.logger)
		if err != nil {
			logger.Errorw("Cannot get state storage for org", "error", err)

			continue
		}

		instance, err := state.FindOne(ctx, instanceID)
		if err != nil {
			if !errors.Is(err, statestorage.ErrInstanceNotFound) {
				logger.Errorw("Cannot find instance in maintenance DB", "error", err)
			}

			continue
		}

		return *instance, nil
	}

	return domain.GetInstanceDetailsSpec{}, errors.New("cannot find instance in maintenance DB(s): no instances found")
}

// LastOperation should fetch the state of the provision/deprovision
// of a cluster.
func (b Broker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (resp domain.LastOperation, err error) {
	logger := b.funcLogger().With("instance_id", instanceID)
	logger.Infow("Fetching state of last operation", "details", details)

	resp.State = domain.Failed

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		return
	}

	cluster, r, err := client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		if r.StatusCode != http.StatusNotFound {
			err = errors.Wrap(err, "cannot get existing cluster")
			logger.Errorw("Failed to get existing cluster", "error", err)

			return
		}
	}

	logger.Infow("Found existing cluster", "cluster", cluster)

	peEndpoints := []*mongodbatlas.PrivateEndpointConnection{}
	for _, pe := range p.PrivateEndpoints {
		if pe.ID == "" {
			continue
		}
		peConnection, _, err := client.PrivateEndpoints.Get(ctx, p.Project.ID, pe.Provider, pe.ID)
		if err != nil {
			logger.Errorw("Failed to fetch Private Endpoint Service Connection from Atlas", "error", err)

			continue
		}
		peEndpoints = append(peEndpoints, peConnection)
	}

	logger.Infow("Found existing Private Endpoints", "endpoints", peEndpoints)

	// brokerapi will NOT update service state if we return any error, so... we won't?
	defer func() {
		if err != nil {
			resp.State = domain.Failed
			resp.Description = "got error: " + err.Error()
			err = nil
		}
	}()

	switch details.OperationData {
	case operationProvision, operationUpdate:
		if r.StatusCode == http.StatusNotFound {
			resp.State = domain.Failed
			resp.Description = "cluster not found"

			return
		}

		retry := false
		logger.Debugw("Create resources", "plan", p)
		retry, err = b.postCreateResources(ctx, client, p)
		if err != nil {
			logger.Debugw("Create resources error", "error", err, "retry", retry)

			break
		}

		if retry {
			resp.State = domain.InProgress
			resp.Description = "resources are being created"

			break
		}

		switch cluster.StateName {
		// Provision has succeeded if the cluster is in state "idle".
		case "IDLE":
			resp.State = domain.Succeeded
		case "CREATING", "UPDATING", "REPAIRING":
			resp.State = domain.InProgress
			resp.Description = cluster.StateName
		default:
			resp.Description = fmt.Sprintf("unknown cluster state %q", cluster.StateName)
		}

	case operationDeprovision:
		switch {
		// The Atlas API may return a 404 response if a cluster is deleted or it
		// will return the cluster with a state of "DELETED". Both of these
		// scenarios indicate that a cluster has been successfully deleted.
		case r.StatusCode == http.StatusNotFound, cluster.StateName == "DELETED":
			if len(peEndpoints) != 0 {
				resp.State = domain.InProgress
				for _, peConnection := range peEndpoints {
					if _, err := client.PrivateEndpoints.Delete(ctx, p.Project.ID, "AZURE", peConnection.ID); err != nil {
						logger.Errorw("Failed to delete Private Endpoint Service from Atlas", "error", err, "connection", peConnection)
					}
				}

				break
			}

			if r.StatusCode == http.StatusNotFound || cluster.StateName == "DELETED" {
				resp.State = domain.Succeeded
			}

			var r *mongodbatlas.Response
			r, err = client.Projects.Delete(ctx, p.Project.ID)
			if err != nil {
				logger.Errorw(
					"Cannot delete Atlas Project",
					"error", err,
					"projectID", p.Project.ID,
					"projectName", p.Project.Name,
				)

				if r.StatusCode != http.StatusNotFound {
					err = errors.Wrap(err, "cannot delete Atlas project")

					break
				}

				// don't fail if the project is already deleted
				err = nil
			}

			state, errDel := b.getState(ctx, p.Project.OrgID)
			if errDel != nil {
				logger.Errorw("Failed to get state storage", "error", errDel)

				break
			}

			errDel = state.DeleteOne(ctx, instanceID)
			if errDel != nil {
				logger.Errorw("Failed to clean up instance from maintenance store", "error", errDel)

				break
			}

		case cluster.StateName == "DELETING":
			resp.State = domain.InProgress

		default:
			resp.Description = fmt.Sprintf("unknown cluster state %q", cluster.StateName)
		}

	default:
		resp.Description = fmt.Sprintf("unknown operation %q", details.OperationData)
	}

	return resp, err
}
