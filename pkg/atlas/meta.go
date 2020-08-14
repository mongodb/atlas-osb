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

package atlas

import (
	"fmt"
	"net/http"
)

// Provider represents a single cloud provider to which a cluster can be
// deployed.
type Provider struct {
	Name          string `json:"@provider"`
	InstanceSizes map[string]InstanceSize
}

// InstanceSize represents an available cluster size.
type InstanceSize struct {
	Name string `json:"name"`
}

// GetProvider will find a provider by name using the private API.
// GET /cloudProviders/{NAME}/options
func (c *HTTPClient) GetProvider(name string) (*Provider, error) {
	path := fmt.Sprintf("cloudProviders/%s/options", name)
	var provider Provider

	err := c.requestPrivate(http.MethodGet, path, nil, &provider)
	return &provider, err
}
