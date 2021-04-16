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

package mongodbrealm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"go.mongodb.org/atlas/mongodbatlas"
)

func (c *Client) RealmAppInputFromString(value string) (*RealmAppInput, error) {
	var t RealmAppInput
	err := json.Unmarshal([]byte(value), &t)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal value")
	}

	return &t, nil
}

type RealmAuth struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	DeviceID     string `json:"device_id,omitempty"`
}

// RealmService is an interface for interfacing with the Realm
// endpoints of the MongoDB Atlas API.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys/
type RealmAppsService interface {
	List(context.Context, string, *ListOptions) ([]RealmApp, *Response, error)
	Get(context.Context, string, string) (*RealmApp, *Response, error)
	Create(context.Context, string, *RealmAppInput) (*RealmApp, *Response, error)
	Update(context.Context, string, string, *RealmAppInput) (*RealmApp, *Response, error)
	Delete(context.Context, string, string) (*Response, error)
}

// RealmAppsServiceOp handles communication with the RealmApp related methods
// of the MongoDB Atlas API
type RealmAppsServiceOp service

var _ RealmAppsService = &RealmAppsServiceOp{}

// RealmAppInput represents MongoDB API key input request for Create.
type RealmAppInput struct {
	Name            string `json:"name,omitempty"`
	ClientAppID     string `json:"client_app_id,omitempty"`
	Location        string `json:"location,omitempty"`
	DeploymentModel string `json:"deployment_model,omitempty"`
	Product         string `json:"product,omitempty"`
}

// RealmApp represents MongoDB API Key.
// {"_id":"5f12de8c15049be9464eb269","client_app_id":"mad-elion-arays","name":"mad-elion","location":"US-VA","deployment_model":"GLOBAL","domain_id":"5f12de8c15049be9464eb26a","group_id":"5f12d8cc6c2bfd1e0c670f4a","last_used":1595072140,"last_modified":1595072140,"product":"standard"}
type RealmApp struct {
	Name            string `json:"name,omitempty"`
	ID              string `json:"_id,omitempty"`
	ClientAppID     string `json:"client_app_id,omitempty"`
	Location        string `json:"location,omitempty"`
	DeploymentModel string `json:"deployment_model,omitempty"`
	GroupID         string `json:"group_id,omitempty"`
	Product         string `json:"product,omitempty"`
	DomainID        string `json:"domain_id,omitempty"`
}

// List all API-KEY in the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-get-all/
func (s *RealmAppsServiceOp) List(ctx context.Context, groupID string, listOptions *ListOptions) ([]RealmApp, *Response, error) {
	path := fmt.Sprintf(realmAppsPath, groupID)

	// Add query params from listOptions
	path, err := setListOptions(path, listOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot set list options")
	}

	req, err := s.Client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create request")
	}

	// root := new(realmAppsResponse)
	root := make([]RealmApp, 0)
	resp, err := s.Client.Do(ctx, req, &root)
	if err != nil {
		return nil, resp, errors.Wrap(err, "cannot do request")
	}

	return root, resp, nil
}

// Get gets the RealmApp specified to {API-KEY-ID} from the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-get-one/
func (s *RealmAppsServiceOp) Get(ctx context.Context, groupID string, appID string) (*RealmApp, *Response, error) {
	if appID == "" {
		return nil, nil, mongodbatlas.NewArgError("appID", "must be set")
	}

	basePath := fmt.Sprintf(realmAppsPath, groupID)
	escapedEntry := url.PathEscape(appID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create request")
	}

	root := new(RealmApp)
	resp, err := s.Client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, errors.Wrap(err, "cannot do request")
	}

	return root, resp, nil
}

// Create an API Key by the {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-create-one/
func (s *RealmAppsServiceOp) Create(ctx context.Context, groupID string, createRequest *RealmAppInput) (*RealmApp, *Response, error) {
	if createRequest == nil {
		return nil, nil, mongodbatlas.NewArgError("createRequest", "cannot be nil")
	}

	path := fmt.Sprintf(realmAppsPath, groupID)

	req, err := s.Client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create request")
	}

	root := new(RealmApp)
	resp, err := s.Client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, errors.Wrap(err, "cannot do request")
	}

	return root, resp, nil
}

// Update a API Key in the organization associated to {ORG-ID}
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-update-one/
func (s *RealmAppsServiceOp) Update(ctx context.Context, groupID, appID string, updateRequest *RealmAppInput) (*RealmApp, *Response, error) {
	if updateRequest == nil {
		return nil, nil, mongodbatlas.NewArgError("updateRequest", "cannot be nil")
	}

	basePath := fmt.Sprintf(realmAppsPath, groupID)
	escapedEntry := url.PathEscape(appID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create request")
	}

	root := new(RealmApp)
	resp, err := s.Client.Do(ctx, req, root)

	return root, resp, errors.Wrap(err, "cannot do request")
}

// Delete the API Key specified to {API-KEY-ID} from the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKey-delete-one-apiKey/
func (s *RealmAppsServiceOp) Delete(ctx context.Context, groupID, appID string) (*Response, error) {
	if appID == "" {
		return nil, mongodbatlas.NewArgError("appID", "must be set")
	}

	basePath := fmt.Sprintf(realmAppsPath, groupID)
	escapedEntry := url.PathEscape(appID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create request")
	}

	resp, err := s.Client.Do(ctx, req, nil)

	return resp, errors.Wrap(err, "cannot do request")
}
