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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/mongodb/atlas-osb/pkg/broker/statestorage"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
)

// The different async operations that can be performed.
// These constants are returned during provisioning, deprovisioning, and
// updates and are subsequently included in async polls from the platform.
const (
	OperationProvision   = "provision"
	OperationDeprovision = "deprovision"
	OperationUpdate      = "update"
	InstanceSizeNameM2   = "M2"
	InstanceSizeNameM5   = "M5"
)

// Provision will create a new Atlas cluster with the instance ID as its name.
// The process is always async.
func (b Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	logger := b.funcLogger()

	logger.Infow("Provisioning instance", "instance_id", instanceID, "details", details)

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
		var newp *mongodbatlas.Project
		newp, err = b.createResources(ctx, client, dp)
		if err != nil {
			return
		}

		dp.Project.ID = newp.ID
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

	state, err := b.stateStorage(dp.Project.OrgID)
	if err != nil {
		return
	}

	v, err := state.Put(ctx, instanceID, s)
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

	logger.Infow("Successfully started Atlas creation process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: OperationProvision,
		DashboardURL:  b.GetDashboardURL(dp.Project.ID, resultingCluster.Name),
	}, nil
}

func (b *Broker) createResources(ctx context.Context, client *mongodbatlas.Client, dp *dynamicplans.Plan) (*mongodbatlas.Project, error) {
	logger := b.funcLogger()

	p, _, err := client.Projects.Create(ctx, dp.Project)
	if err != nil {
		logger.Errorw("Cannot create project", "error", err, "project", dp.Project)
		return nil, err
	}

	for _, u := range dp.DatabaseUsers {
		_, _, err := client.DatabaseUsers.Create(ctx, p.ID, u)
		if err != nil {
			return nil, err
		}
	}

	if len(dp.IPWhitelists) > 0 {
		_, _, err := client.ProjectIPWhitelist.Create(ctx, p.ID, dp.IPWhitelists)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// Update will change the configuration of an existing Atlas cluster asynchronously.
func (b Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (spec domain.UpdateServiceSpec, err error) {
	logger := b.funcLogger()
	logger.Infow("Updating instance", "instance_id", instanceID, "details", details)

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
		request := &mongodbatlas.Cluster{
			Paused: &paused,
		}

		_, _, err = client.Clusters.Update(ctx, oldPlan.Project.ID, oldPlan.Cluster.Name, request)
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

	newPlan, err := b.parsePlan(planContext, details.PlanID)
	if err != nil {
		return
	}

	// Atlas doesn't allow for cluster renaming - ignore any changes
	newPlan.Cluster.Name = existingCluster.Name

	resultingCluster, _, err := client.Clusters.Update(ctx, oldPlan.Project.ID, existingCluster.Name, newPlan.Cluster)
	if err != nil {
		logger.Errorw("Failed to update Atlas cluster", "error", err, "new_cluster", newPlan.Cluster)
		return
	}

	planEnc, err := encodePlan(*oldPlan)
	if err != nil {
		return
	}

	oldPlan.Cluster = resultingCluster
	s := domain.GetInstanceDetailsSpec{
		PlanID:       details.PlanID,
		ServiceID:    details.ServiceID,
		DashboardURL: b.GetDashboardURL(oldPlan.Project.ID, oldPlan.Cluster.Name),
		Parameters:   planEnc,
	}

	state, err := b.stateStorage(oldPlan.Project.OrgID)
	if err != nil {
		return
	}

	// TODO: make this error-out reversible?
	err = state.DeleteOne(ctx, instanceID)
	if err != nil {
		logger.Errorw("Error delete from state", "err", err, "instanceID", instanceID)
		return
	}

	obj, err := state.Put(ctx, instanceID, s)
	if err != nil {
		logger.Errorw("Error insert one from state", "err", err, "instanceID", instanceID, "s", s)
		return
	}
	//
	//s, err := b.state.UpdateOne(instanceID,
	logger.Infow("Inserted into state", "obj", obj)
	logger.Infow("Successfully started Atlas cluster update process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: OperationUpdate,
		DashboardURL:  b.GetDashboardURL(oldPlan.Project.ID, resultingCluster.Name),
	}, nil
}

// Deprovision will destroy an Atlas cluster asynchronously.
func (b Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (spec domain.DeprovisionServiceSpec, err error) {
	logger := b.funcLogger()
	logger.Infow("Deprovisioning instance", "instance_id", instanceID, "details", details)

	client, p, err := b.getClient(ctx, instanceID, details.PlanID, nil)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	_, err = client.Clusters.Delete(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil {
		logger.Errorw("Failed to delete Atlas cluster", "error", err, "instance_id", instanceID)
		return
	}

	for _, u := range p.DatabaseUsers {
		_, err = client.DatabaseUsers.Delete(ctx, u.DatabaseName, p.Project.ID, u.Username)
		if err != nil {
			logger.Errorw("failed to delete Database user", "error", err, "username", u.Username)
		}
	}

	logger.Infow("Successfully started Atlas cluster deletion process", "instance_id", instanceID)

	return domain.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: OperationDeprovision,
	}, nil
}

// GetInstance should fetch the stored instance from state storage
func (b Broker) GetInstance(ctx context.Context, instanceID string) (spec domain.GetInstanceDetailsSpec, err error) {
	logger := b.funcLogger().With("instanceID", instanceID)
	logger.Info("Fetching instance")

	err = b.getState(ctx, instanceID, &spec)
	if err != nil {
		logger.Errorw("Unable to fetch instance", "err", err)
		return spec, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "get-instance")
	}

	return spec, nil
}

func (b Broker) getState(ctx context.Context, id string, out interface{}) error {
	logger := b.funcLogger().With("id", id)

	for k := range b.credentials.Keys() {
		logger = logger.With("orgID", k)

		ss, err := b.stateStorage(k)
		if err != nil {
			logger.Errorw("Cannot get state storage for org", "error", err)
			continue
		}

		var value string
		err = ss.FindOne(ctx, id, &value)
		if err != nil {
			if err != statestorage.ErrInstanceNotFound {
				logger.Errorw("Cannot find instance in maintenance DB", "error", err)
			}
			continue
		}

		b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(value))
		err = json.NewDecoder(b64).Decode(out)
		return err
	}

	return errors.New("cannot find state in maintenance DB(s): no state found")
}

// LastOperation should fetch the state of the provision/deprovision
// of a cluster.
func (b Broker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (resp domain.LastOperation, err error) {
	logger := b.funcLogger()
	logger.Infow("Fetching state of last operation", "instance_id", instanceID, "details", details)

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

	cluster, r, err := client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil && r.StatusCode != http.StatusNotFound {
		err = errors.Wrap(err, "cannot get existing cluster")
		logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		return
	}

	logger.Infow("Found existing cluster", "cluster", cluster)

	switch details.OperationData {
	case OperationProvision, OperationUpdate:
		if r.StatusCode == http.StatusNotFound {
			resp.State = domain.Failed
			resp.Description = "cluster not found"
			return
		}

		switch cluster.StateName {
		// Provision has succeeded if the cluster is in state "idle".
		case "IDLE":
			resp.State = domain.Succeeded
		case "CREATING", "UPDATING":
			resp.State = domain.InProgress
		default:
			resp.Description = fmt.Sprintf("unknown cluster state %q", cluster.StateName)
		}

	case OperationDeprovision:
		switch {
		// The Atlas API may return a 404 response if a cluster is deleted or it
		// will return the cluster with a state of "DELETED". Both of these
		// scenarios indicate that a cluster has been successfully deleted.
		case r.StatusCode == http.StatusNotFound, cluster.StateName == "DELETED":
			if r.StatusCode == http.StatusNotFound || cluster.StateName == "DELETED" {
				resp.State = domain.Succeeded
			}

			_, err = client.Projects.Delete(ctx, p.Project.ID)
			if err != nil {
				err = errors.Wrap(err, "cannot delete Atlas project")
				logger.Errorw(
					"Cannot delete Atlas Project",
					"error", err,
					"projectID", p.Project.ID,
					"projectName", p.Project.Name,
				)
			}

			state, errDel := b.stateStorage(p.Project.OrgID)
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
	}

	return resp, err
}

// NormalizeClusterName will sanitize a name to make sure it will be accepted
// by the Atlas API. Atlas has different name length requirements depending on
// which environment it's running in. A length of 23 is a safe choice and
// truncates UUIDs nicely.
func NormalizeClusterName(name string) string {
	const maximumNameLength = 23

	if len(name) > maximumNameLength {
		return name[0:maximumNameLength]
	}

	return name
}
