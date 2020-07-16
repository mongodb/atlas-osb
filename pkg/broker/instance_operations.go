package broker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"go.mongodb.org/mongo-driver/bson"
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
	b.logger.Infow("Provisioning instance", "instance_id", instanceID, "details", details)

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

	client, p, dp, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	if p.ID == "" {
		var newp *mongodbatlas.Project
		newp, err = b.createResources(ctx, client, dp)
		if err != nil {
			return
		}

		p.ID = newp.ID
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	// Construct a cluster definition from the instance ID, service, plan, and params.
	b.logger.Infow("Creating cluster", "instance_name", planContext["instance_name"])
	// TODO - add this context info about k8s/namespace or pcf space into labels

	s := serviceInstance{
		ID: instanceID,
		GetInstanceDetailsSpec: domain.GetInstanceDetailsSpec{
			PlanID:       details.PlanID,
			ServiceID:    details.ServiceID,
			DashboardURL: b.GetDashboardURL(p.ID, dp.Cluster.Name),
			Parameters: bson.M{
				"groupID": p.ID,
				"plan":    *dp,
			},
		},
	}

	if b.client != nil {
		col := b.client.Database("atlas-broker").Collection("instances")
		_, err = col.InsertOne(ctx, s)
		if err != nil {
			return
		}

		defer func() {
			if err != nil {
				_, _ = col.DeleteOne(ctx, s)
			}
		}()
	}

	// Create a new Atlas cluster from the generated definition
	resultingCluster, _, err := client.Clusters.Create(ctx, p.ID, dp.Cluster)

	if err != nil {
		b.logger.Errorw("Failed to create Atlas cluster", "error", err, "cluster", dp.Cluster)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully started Atlas creation process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: OperationProvision,
		DashboardURL:  b.GetDashboardURL(p.ID, resultingCluster.Name),
	}, nil
}

func (b *Broker) createResources(ctx context.Context, client *mongodbatlas.Client, dp *dynamicplans.Plan) (*mongodbatlas.Project, error) {
	p, _, err := client.Projects.Create(ctx, dp.Project)
	if err != nil {
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

	b.credentials.Projects[p.ID] = b.credentials.Orgs[p.OrgID]
	return p, nil
}

// Update will change the configuration of an existing Atlas cluster asynchronously.
func (b Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (spec domain.UpdateServiceSpec, err error) {
	b.logger.Infow("Updating instance", "instance_id", instanceID, "details", details)

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

	client, p, oldPlan, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
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

		_, _, err = client.Clusters.Update(ctx, p.ID, oldPlan.Cluster.Name, request)
		return
	}

	// Fetch the cluster from Atlas. The Atlas API requires an instance size to
	// be passed during updates (if there are other update to the provider, such
	// as region). The plan is not included in the OSB call unless it has changed
	// hence we need to fetch the current value from Atlas.
	existingCluster, _, err := client.Clusters.Get(ctx, p.ID, oldPlan.Cluster.Name)
	if err != nil {
		err = atlasToAPIError(err)
		return
	}

	newPlan, err := b.parsePlan(planContext, details.PlanID)
	if err != nil {
		return
	}

	resultingCluster, _, err := client.Clusters.Update(ctx, p.ID, existingCluster.Name, newPlan.Cluster)
	if err != nil {
		b.logger.Errorw("Failed to update Atlas cluster", "error", err, "new_cluster", newPlan.Cluster)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully started Atlas cluster update process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: OperationUpdate,
		DashboardURL:  b.GetDashboardURL(p.ID, resultingCluster.Name),
	}, nil
}

// Deprovision will destroy an Atlas cluster asynchronously.
func (b Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (spec domain.DeprovisionServiceSpec, err error) {
	b.logger.Infow("Deprovisioning instance", "instance_id", instanceID, "details", details)

	p, err := b.getInstancePlan(ctx, instanceID)
	if err != nil {
		return
	}

	client, err := b.createClient(b.credentials.Projects[p.Project.ID])
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
		b.logger.Errorw("Failed to delete Atlas cluster", "error", err, "instance_id", instanceID)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully started Atlas cluster deletion process", "instance_id", instanceID)

	return domain.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: OperationDeprovision,
	}, nil
}

// GetInstance is currently not supported as specified by the
// InstancesRetrievable setting in the service catalog.
func (b Broker) GetInstance(ctx context.Context, instanceID string) (spec domain.GetInstanceDetailsSpec, err error) {
	b.logger.Infow("Fetching instance", "instance_id", instanceID)

	err = b.client.
		Database("atlas-broker").
		Collection("instances").
		FindOne(ctx, bson.M{"id": instanceID}).
		Decode(&spec)
	return
}

// LastOperation should fetch the state of the provision/deprovision
// of a cluster.
func (b Broker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (resp domain.LastOperation, err error) {
	b.logger.Infow("Fetching state of last operation", "instance_id", instanceID, "details", details)

	p, err := b.getInstancePlan(ctx, instanceID)
	if err != nil {
		return
	}

	client, err := b.createClient(b.credentials.Projects[p.Project.ID])
	if err != nil {
		return
	}

	cluster, r, err := client.Clusters.Get(ctx, p.Project.ID, p.Cluster.Name)
	if err != nil && r.StatusCode != http.StatusNotFound {
		b.logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Found existing cluster", "cluster", cluster)

	state := domain.LastOperationState(domain.Failed)

	switch details.OperationData {
	case OperationProvision, OperationUpdate:
		if r.StatusCode == http.StatusNotFound {
			state = domain.Failed
			break
		}

		switch cluster.StateName {
		// Provision has succeeded if the cluster is in state "idle".
		case "IDLE":
			state = domain.Succeeded
		case "CREATING", "UPDATING":
			state = domain.InProgress
		}
	case OperationDeprovision:
		// The Atlas API may return a 404 response if a cluster is deleted or it
		// will return the cluster with a state of "DELETED". Both of these
		// scenarios indicate that a cluster has been successfully deleted.
		if r.StatusCode == http.StatusNotFound || cluster.StateName == "DELETED" {
			state = domain.Succeeded
			if b.client != nil {
				// TODO: change this?
				_, _ = b.client.
					Database("atlas-broker").
					Collection("instances").
					DeleteOne(ctx, bson.M{"id": instanceID})
			}
		} else if cluster.StateName == "DELETING" {
			state = domain.InProgress
		}
	}

	return domain.LastOperation{
		State: state,
	}, nil
}

// NormalizeClusterName will sanitize a name to make sure it will be accepted
// by the Atlas API. Atlas has different name length requirements depending on
// which environment it's running in. A length of 23 is a safe choice and
// truncates UUIDs nicely.
func NormalizeClusterName(name string) string {
	const maximumNameLength = 23

	if len(name) > maximumNameLength {
		return string(name[0:maximumNameLength])
	}

	return name
}
