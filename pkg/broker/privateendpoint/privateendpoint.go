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

package privateendpoint

import (
	"context"
	"net/http"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-11-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	"go.mongodb.org/atlas/mongodbatlas"
)

type (
	Provider = string
	Region   = string
)

type PrivateEndpoints map[Provider]map[Region]*EndpointService

type EndpointService struct {
	ID        string             `json:"serviceID,omitempty"`
	Endpoints []*PrivateEndpoint `json:"endpoints,omitempty"`
}

type PrivateEndpoint struct {
	SubscriptionID     string `json:"subscriptionID,omitempty"`
	ResourceGroup      string `json:"resourceGroup,omitempty"`
	VirtualNetworkName string `json:"virtualNetworkName,omitempty"`
	SubnetName         string `json:"subnetName,omitempty"`
	EndpointName       string `json:"endpointName,omitempty"`
}

func Create(ctx context.Context, e *PrivateEndpoint, pe *mongodbatlas.PrivateEndpointConnection) (futureWrapper func() (network.PrivateEndpoint, error), err error) {
	// create an authorizer from env vars or Azure Managed Service Idenity
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create authorizer from environment")
	}

	// disable network policies for Private Endpoints: https://docs.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy

	snClient := network.NewSubnetsClient(e.SubscriptionID)
	snClient.Authorizer = authorizer

	// update request has to contain AddressPrefix, so we have to retrieve it first
	// also we need the subnet info during PE creation
	sn, err := snClient.Get(ctx, e.ResourceGroup, e.VirtualNetworkName, e.SubnetName, "")
	if err != nil {
		return nil, errors.Wrap(err, "cannot get existing subnet")
	}

	sn.PrivateEndpointNetworkPolicies = to.StringPtr("Disabled")

	_, err = snClient.CreateOrUpdate(ctx, e.ResourceGroup, e.VirtualNetworkName, e.SubnetName, sn)
	if err != nil {
		return nil, errors.Wrap(err, "cannot update subnet")
	}

	// create the Private Endpoint
	peClient := network.NewPrivateEndpointsClient(e.SubscriptionID)
	peClient.Authorizer = authorizer

	ep, err := peClient.Get(ctx, e.ResourceGroup, e.EndpointName, "")
	if err != nil {
		if ep.StatusCode != http.StatusNotFound {
			return nil, errors.Wrap(err, "cannot retrieve private endpoint")
		}
	} else {
		// TODO: handle existing endpoint properly
		return func() (network.PrivateEndpoint, error) {
			return ep, nil
		}, nil
	}

	future, err := peClient.CreateOrUpdate(ctx, e.ResourceGroup, e.EndpointName, network.PrivateEndpoint{
		Location: to.StringPtr("westus"), // TODO: add another field to plan? or deduce from Atlas region name?
		PrivateEndpointProperties: &network.PrivateEndpointProperties{
			Subnet: &sn,

			// TODO: should we use PrivateLinkServiceConnections instead?
			ManualPrivateLinkServiceConnections: &[]network.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr(pe.PrivateLinkServiceName),

					PrivateLinkServiceConnectionProperties: &network.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(pe.PrivateLinkServiceResourceID),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create or update endpoint")
	}

	return func() (network.PrivateEndpoint, error) {
		return future.Result(peClient)
	}, nil
}

func GetIPAddress(ctx context.Context, azurePE network.PrivateEndpoint, e *PrivateEndpoint) (string, error) {
	if azurePE.NetworkInterfaces == nil || len(*azurePE.NetworkInterfaces) == 0 {
		return "", errors.New("no NetworkInterfaces in endpoint")
	}

	// create an authorizer from env vars or Azure Managed Service Idenity
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return "", errors.Wrap(err, "cannot create authorizer from environment")
	}

	i := (*azurePE.NetworkInterfaces)[0]

	ifClient := network.NewInterfacesClient(e.SubscriptionID)
	ifClient.Authorizer = authorizer

	// only ID is included in the response
	// name is the last element of the resource ID by default
	// TODO: verify this doesn't change
	i, err = ifClient.Get(ctx, e.ResourceGroup, path.Base(*i.ID), "")
	if err != nil {
		return "", errors.Wrap(err, "cannot get network interface")
	}

	if i.InterfacePropertiesFormat == nil || i.IPConfigurations == nil || len(*i.IPConfigurations) == 0 {
		return "", errors.New("no IPConfigurations in NetworkInterface associated with endpoint")
	}

	// TODO: ipConfiguration where name == 'privateEndpointIpConfig'
	conf := (*i.IPConfigurations)[0]

	if conf.PrivateIPAddress == nil {
		return "", errors.New("nil IPAddress in NetworkInterface/IPConfiguration associated with endpoint")
	}

	return *conf.PrivateIPAddress, nil
}
