package broker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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

type ContextParams struct {
	InstanceName string `json:"instance_name"`
	Namespace    string `json:"namespace"`
	Platform     string `json:"platform"`
}

// Provision will create a new Atlas cluster with the instance ID as its name.
// The process is always async.
func (b Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	b.logger.Infow("Provisioning instance", "instance_id", instanceID, "details", details)

	client, gid, err := b.getClient(ctx, details.PlanID, details.RawParameters)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	// Construct a cluster definition from the instance ID, service, plan, and params.
	contextParams := &ContextParams{}
	_ = json.Unmarshal(details.RawContext, contextParams)
	b.logger.Infow("Creating cluster", "instance_name", contextParams.InstanceName)
	// TODO - add this context info about k8s/namespace or pcf space into labels
	cluster, err := b.clusterFromParams(instanceID, details.ServiceID, details.PlanID, details.RawParameters)
	if err != nil {
		b.logger.Errorw("Couldn't create cluster from the passed parameters", "error", err, "instance_id", instanceID, "details", details)
		return
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

// Update will change the configuration of an existing Atlas cluster asynchronously.
func (b Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (spec domain.UpdateServiceSpec, err error) {
	b.logger.Infow("Updating instance", "instance_id", instanceID, "details", details)

	client, gid, err := b.getClient(ctx, details.PlanID, details.RawParameters)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	// Fetch the cluster from Atlas. The Atlas API requires an instance size to
	// be passed during updates (if there are other update to the provider, such
	// as region). The plan is not included in the OSB call unless it has changed
	// hence we need to fetch the current value from Atlas.
	existingCluster, _, err := client.Clusters.Get(ctx, gid, NormalizeClusterName(instanceID))
	if err != nil {
		err = atlasToAPIError(err)
		return
	}

	// Construct a cluster from the instance ID, service, plan, and params.
	contextParams := &ContextParams{}
	_ = json.Unmarshal(details.RawContext, contextParams)

	cluster, err := b.clusterFromParams(instanceID, details.ServiceID, details.PlanID, details.RawParameters)
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

	client, gid, err := b.getClient(ctx, details.PlanID, nil)
	if err != nil {
		return
	}

	// Async needs to be supported for provisioning to work.
	if !asyncAllowed {
		err = apiresponses.ErrAsyncRequired
		return
	}

	_, err = client.Clusters.Delete(ctx, gid, NormalizeClusterName(instanceID))
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
		err = apiresponses.NewFailureResponse(errors.New("Fetching instances not supported in stateless mode"), http.StatusNotImplemented, "get-instance")
		return
	}

	err = b.client.Ping(ctx, readpref.Primary())
	if err != nil {
		return
	}

	c := b.client.Database("atlas-broker").Collection("instances")
	c.FindOne(ctx, bson.M{"id": instanceID})
	s := serviceInstance{}
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

	client, gid, err := b.getClient(ctx, details.PlanID, nil)
	if err != nil {
		return
	}

	cluster, _, err := client.Clusters.Get(ctx, gid, NormalizeClusterName(instanceID))
	if err != nil && err != atlas.ErrClusterNotFound {
		b.logger.Errorw("Failed to get existing cluster", "error", err, "instance_id", instanceID)
		err = atlasToAPIError(err)
		return
	}

	b.logger.Infow("Found existing cluster", "cluster", cluster)

	state := domain.LastOperationState(domain.Failed)

	switch details.OperationData {
	case OperationProvision:
		switch cluster.StateName {
		// Provision has succeeded if the cluster is in state "idle".
		case atlas.ClusterStateIdle:
			state = domain.Succeeded
		case atlas.ClusterStateCreating:
			state = domain.InProgress
		}
	case OperationDeprovision:
		// The Atlas API may return a 404 response if a cluster is deleted or it
		// will return the cluster with a state of "DELETED". Both of these
		// scenarios indicate that a cluster has been successfully deleted.
		if err == atlas.ErrClusterNotFound || cluster.StateName == atlas.ClusterStateDeleted {
			state = domain.Succeeded
		} else if cluster.StateName == atlas.ClusterStateDeleting {
			state = domain.InProgress
		}
	case OperationUpdate:
		// We assume that the cluster transitions to the "UPDATING" state
		// in a synchronous manner during the update request.
		switch cluster.StateName {
		case atlas.ClusterStateIdle:
			state = domain.Succeeded
		case atlas.ClusterStateUpdating:
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
func (b Broker) clusterFromParams(instanceID string, serviceID string, planID string, rawParams []byte) (*mongodbatlas.Cluster, error) {
	// Set up a params object which will be used for deserialiation.
	params := dynamicplans.DefaultCtx(b.credentials)

	// If params were passed we unmarshal them into the params object.
	if len(rawParams) > 0 {
		err := json.Unmarshal(rawParams, &params)
		if err != nil {
			return nil, err
		}
	}

	// If the plan ID is specified we construct the provider object from the service and plan.
	// The plan ID is optional during updates but not during creation.
	if planID != "" {
		if params.Cluster.ProviderSettings == nil {
			params.Cluster.ProviderSettings = &mongodbatlas.ProviderSettings{}
		}

		instanceSizeName := params.Cluster.ProviderSettings.InstanceSizeName
		if instanceSizeName != InstanceSizeNameM2 && instanceSizeName != InstanceSizeNameM5 {
			provider, err := b.catalog.findProviderByServiceID(serviceID)
			if err != nil {
				return nil, err
			}

			instanceSize, err := b.catalog.findInstanceSizeByPlanID(provider, planID)
			if err != nil {
				return nil, err
			}

			// Configure provider based on service and plan.
			params.Cluster.ProviderSettings.ProviderName = provider.Name
			params.Cluster.ProviderSettings.InstanceSizeName = instanceSize.Name
		}
	}

	// Add the instance ID as the name of the cluster.
	if params.Cluster.Name == "" {
		params.Cluster.Name = NormalizeClusterName(instanceID)
	}
	return params.Cluster, nil
}
