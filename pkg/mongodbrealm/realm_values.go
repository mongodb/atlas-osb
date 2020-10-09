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

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

const (
	realmValuesPath = "groups/%s/apps/%s/values"
)

func (c *Client) RealmValueFromString(key string, value string) (*RealmValue, error) {
	v := &RealmValue{
		ID:    key,
		Name:  key,
		Value: json.RawMessage(value), // ????
	}
	return v, nil
}

// RealmService is an interface for interfacing with the Realm

// endpoints of the MongoDB Atlas API.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys/
type RealmValuesService interface {
	List(context.Context, string, string, *ListOptions) ([]RealmValue, *Response, error)
	Get(context.Context, string, string, string) (*RealmValue, *Response, error)
	Create(context.Context, string, string, *RealmValue) (*RealmValue, *Response, error)
	Update(context.Context, string, string, string, *RealmValue) (*RealmValue, *Response, error)
	Delete(context.Context, string, string, string) (*Response, error)
}

// RealmValuesServiceOp handles communication with the RealmValue related methods
// of the MongoDB Atlas API
type RealmValuesServiceOp service

var _ RealmValuesService = &RealmValuesServiceOp{}

type RealmValue struct {
	ID      string          `json:"_id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Value   json.RawMessage `json:"value,omitempty"`
	Private bool            `json:"private,omitempty"`
}

// realmValuesResponse is the response from the RealmValuesService.List.
// type realmValuesResponse struct {
//        Apps []RealmValue
// }

// List all API-KEY in the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-get-all/
func (s *RealmValuesServiceOp) List(ctx context.Context, groupID string, appID string, listOptions *ListOptions) ([]RealmValue, *Response, error) {
	path := fmt.Sprintf(realmValuesPath, groupID, appID)

	// Add query params from listOptions
	path, err := setListOptions(path, listOptions)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.Client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := make([]RealmValue, 0)
	resp, err := s.Client.Do(ctx, req, &root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Get gets the RealmValue specified to {API-KEY-ID} from the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-get-one/
func (s *RealmValuesServiceOp) Get(ctx context.Context, groupID string, appID string, valueID string) (*RealmValue, *Response, error) {
	if appID == "" {
		return nil, nil, mongodbatlas.NewArgError("appID", "must be set")
	}

	basePath := fmt.Sprintf(realmValuesPath, groupID, appID)
	escapedEntry := url.PathEscape(valueID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(RealmValue)
	resp, err := s.Client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

// Create an API Key by the {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-create-one/
func (s *RealmValuesServiceOp) Create(ctx context.Context, groupID string, appID string, createRequest *RealmValue) (*RealmValue, *Response, error) {
	if createRequest == nil {
		return nil, nil, mongodbatlas.NewArgError("createRequest", "cannot be nil")
	}

	path := fmt.Sprintf(realmValuesPath, groupID, appID)

	req, err := s.Client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(RealmValue)
	resp, err := s.Client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

// Update a API Key in the organization associated to {ORG-ID}
// See more: https://docs.atlas.mongodb.com/reference/api/apiKeys-orgs-update-one/
func (s *RealmValuesServiceOp) Update(ctx context.Context, groupID string, appID string, keyID string, updateRequest *RealmValue) (*RealmValue, *Response, error) {
	if updateRequest == nil {
		return nil, nil, mongodbatlas.NewArgError("updateRequest", "cannot be nil")
	}

	basePath := fmt.Sprintf(realmValuesPath, groupID, appID)
	escapedEntry := url.PathEscape(keyID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(RealmValue)
	resp, err := s.Client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

// Delete the API Key specified to {API-KEY-ID} from the organization associated to {ORG-ID}.
// See more: https://docs.atlas.mongodb.com/reference/api/apiKey-delete-one-apiKey/
func (s *RealmValuesServiceOp) Delete(ctx context.Context, groupID, appID string, keyID string) (*Response, error) {
	if appID == "" {
		return nil, mongodbatlas.NewArgError("appID", "must be set")
	}

	basePath := fmt.Sprintf(realmValuesPath, groupID, appID)
	escapedEntry := url.PathEscape(keyID)
	path := fmt.Sprintf("%s/%s", basePath, escapedEntry)

	req, err := s.Client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Client.Do(ctx, req, nil)

	return resp, err
}
