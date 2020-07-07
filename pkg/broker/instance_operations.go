package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"gopkg.in/mgo.v2/bson"
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
		"Credentials": b.credentials,
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

	if b.mode == DynamicPlans && gid == "" {
		p := &mongodbatlas.Project{}
		p, err = b.createResources(ctx, client, details.PlanID, planContext)
		if err != nil {
			return
		}

		gid = p.ID
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	// Construct a cluster definition from the instance ID, service, plan, and params.
	b.logger.Infow("Creating cluster", "instance_name", planContext["instance_name"])
	// TODO - add this context info about k8s/namespace or pcf space into labels
	cluster, err := b.clusterFromParams(instanceID, details.ServiceID, details.PlanID, planContext)
	if err != nil {
		b.logger.Errorw("Couldn't create cluster from the passed parameters", "error", err, "instance_id", instanceID, "details", details)
		return
	}

	s := serviceInstance{
		ID: instanceID,
		GetInstanceDetailsSpec: domain.GetInstanceDetailsSpec{
			PlanID:       details.PlanID,
			ServiceID:    details.ServiceID,
			DashboardURL: b.GetDashboardURL(gid, cluster.Name),
			Parameters: bson.M{
				"groupID":     gid,
				"clusterName": cluster.Name,
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
				col.DeleteOne(ctx, s)
			}
		}()
	}

	// Add default labels
	// TODO - append the env info k8s, pcf, etc
	var defaultLabel = mongodbatlas.Label{Key: "Infrastructure Tool", Value: "MongoDB Atlas Service Broker"}
	cluster.Labels = []mongodbatlas.Label{defaultLabel}
	// Create a new Atlas cluster from the generated definition
	resultingCluster, _, err := client.Clusters.Create(ctx, gid, cluster)

	if err != nil {
		b.logger.Errorw("Failed to create Atlas cluster", "error", err, "cluster", cluster)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully started Atlas creation process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: OperationProvision,
		DashboardURL:  b.GetDashboardURL(gid, resultingCluster.Name),
	}, nil
}

func (b *Broker) createResources(ctx context.Context, client *mongodbatlas.Client, planID string, planContext dynamicplans.Context) (*mongodbatlas.Project, error) {
	dp, err := b.parsePlan(planContext, planID)
	if err != nil {
		return nil, err
	}

	if dp.Project == nil {
		return nil, fmt.Errorf("missing Project in plan definition")
	}

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
		"Credentials": b.credentials,
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

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	name, err := b.getClusterNameByInstanceID(ctx, instanceID)
	if err != nil {
		return
	}

	// Fetch the cluster from Atlas. The Atlas API requires an instance size to
	// be passed during updates (if there are other update to the provider, such
	// as region). The plan is not included in the OSB call unless it has changed
	// hence we need to fetch the current value from Atlas.
	existingCluster, _, err := client.Clusters.Get(ctx, gid, name)
	if err != nil {
		err = atlasToAPIError(err)
		return
	}

	// Construct a cluster from the instance ID, service, plan, and params.
	cluster, err := b.clusterFromParams(instanceID, details.ServiceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	// Make sure the cluster provider has all the neccessary params for the
	// Atlas API. The Atlas API requires both the provider name and instance
	// size if the provider object is set. If they are missing we use the
	// existing values.
	if cluster.ProviderSettings != nil {
		if cluster.ProviderSettings.ProviderName == "" {
			cluster.ProviderSettings.ProviderName = existingCluster.ProviderSettings.ProviderName
		}

		if cluster.ProviderSettings.InstanceSizeName == "" {
			cluster.ProviderSettings.InstanceSizeName = existingCluster.ProviderSettings.InstanceSizeName
		}
	}

	resultingCluster, _, err := client.Clusters.Update(ctx, gid, existingCluster.Name, cluster)
	if err != nil {
		b.logger.Errorw("Failed to update Atlas cluster", "error", err, "cluster", cluster)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Successfully started Atlas cluster update process", "instance_id", instanceID, "cluster", resultingCluster)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: OperationUpdate,
		DashboardURL:  b.GetDashboardURL(gid, resultingCluster.Name),
	}, nil
}

// Deprovision will destroy an Atlas cluster asynchronously.
func (b Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (spec domain.DeprovisionServiceSpec, err error) {
	b.logger.Infow("Deprovisioning instance", "instance_id", instanceID, "details", details)

	planContext := dynamicplans.Context{
		"Credentials": b.credentials,
		"instance_id": instanceID,
	}
	client, gid, err := b.getClient(ctx, instanceID, details.PlanID, planContext)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	name, err := b.getClusterNameByInstanceID(ctx, instanceID)
	if err != nil {
		return
	}

	_, err = client.Clusters.Delete(ctx, gid, name)
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

	if b.client == nil {
		err = apiresponses.NewFailureResponse(errors.New("Fetching instances is not supported in stateless mode"), http.StatusNotImplemented, "get-instance")
		return
	}

	c := b.client.Database("atlas-broker").Collection("instances")
	s := serviceInstance{}

	err = c.FindOne(ctx, bson.M{"id": instanceID}).Decode(&s)
	if err != nil {
		return
	}

	return domain.GetInstanceDetailsSpec{
		ServiceID:    s.ServiceID,
		PlanID:       s.PlanID,
		DashboardURL: s.DashboardURL,
		Parameters:   s.Parameters,
	}, nil
}

// LastOperation should fetch the state of the provision/deprovision
// of a cluster.
func (b Broker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (resp domain.LastOperation, err error) {
	b.logger.Infow("Fetching state of last operation", "instance_id", instanceID, "details", details)

	planContext := dynamicplans.Context{
		"Credentials": b.credentials,
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

	cluster, r, err := client.Clusters.Get(ctx, gid, name)
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
				b.client.Database("atlas-broker").Collection("instances").DeleteOne(ctx, bson.M{"id": instanceID})
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

// clusterFromParams will construct a cluster object from an instance ID,
// service, plan, and raw parameters. This way users can pass all the
// configuration available for clusters in the Atlas API as "cluster" in the params.
func (b Broker) clusterFromParams(instanceID string, serviceID string, planID string, planContext dynamicplans.Context) (*mongodbatlas.Cluster, error) {
	// In template mode, everything is handled by the template itself.
	if b.mode == DynamicPlans {
		dp, err := b.parsePlan(planContext, planID)
		return dp.Cluster, err
	}

	// workaround for old modes
	var context struct {
		Cluster *mongodbatlas.Cluster `json:"cluster"`
	}

	out, _ := json.Marshal(planContext)
	_ = json.Unmarshal(out, &context)

	// If the plan ID is specified we construct the provider object from the service and plan.
	// The plan ID is optional during updates but not during creation.
	if planID != "" {
		if context.Cluster.ProviderSettings == nil {
			context.Cluster.ProviderSettings = &mongodbatlas.ProviderSettings{}
		}

		instanceSizeName := context.Cluster.ProviderSettings.InstanceSizeName
		if instanceSizeName != InstanceSizeNameM2 && instanceSizeName != InstanceSizeNameM5 {
			provider, err := b.catalog.findProviderByServiceID(serviceID)
			if err != nil {
				return nil, err
			}

			instanceSize, err := b.catalog.findInstanceSizeByPlanID(planID)
			if err != nil {
				return nil, err
			}

			// Configure provider based on service and plan.
			context.Cluster.ProviderSettings.ProviderName = provider.Name
			context.Cluster.ProviderSettings.InstanceSizeName = instanceSize.Name
		}
	}

	// Add the instance ID as the name of the cluster.
	context.Cluster.Name = NormalizeClusterName(instanceID)
	return context.Cluster, nil
}
