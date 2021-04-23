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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
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
	ID        string `json:"serviceID,omitempty"`
	Endpoints []*PrivateEndpoint
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

	_, err = snClient.CreateOrUpdate(ctx, e.ResourceGroup, e.VirtualNetworkName, e.SubnetName, network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			PrivateEndpointNetworkPolicies: to.StringPtr("Disabled"),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create or update subnet")
	}

	// create the Private Endpoint
	peClient := network.NewPrivateEndpointsClient(e.SubscriptionID)
	peClient.Authorizer = authorizer

	future, err := peClient.CreateOrUpdate(ctx, e.ResourceGroup, e.EndpointName, network.PrivateEndpoint{
		PrivateEndpointProperties: &network.PrivateEndpointProperties{
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

func GetIPAddress(e network.PrivateEndpoint) (string, error) {
	if e.NetworkInterfaces == nil || len(*e.NetworkInterfaces) == 0 {
		return "", errors.New("no NetworkInterfaces in endpoint")
	}

	i := (*e.NetworkInterfaces)[0]

	if i.IPConfigurations == nil || len(*i.IPConfigurations) == 0 {
		return "", errors.New("no IPConfigurations in NetworkInterface associated with endpoint")
	}

	conf := (*i.IPConfigurations)[0]

	if conf.PrivateIPAddress == nil {
		return "", errors.New("nil IPAddress in NetworkInterface/IPConfiguration associated with endpoint")
	}

	return *conf.PrivateIPAddress, nil
}
