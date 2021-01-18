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

	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

type StateStorage interface {
	Put(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) error
	Get(ctx context.Context, key string) (*domain.GetInstanceDetailsSpec, error)
	Update(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) error
	Delete(ctx context.Context, key string) error
}

type mongoStorage struct {
	client *mongo.Client
}

type mongoData struct {
	ID    string                         `bson:"id"`
	Value *domain.GetInstanceDetailsSpec `bson:",omitempty"`
}

func NewMongoStorage(client *mongo.Client) StateStorage {
	return &mongoStorage{client}
}

func (m mongoStorage) Put(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) error {
	_, err := m.client.
		Database("atlas-broker").
		Collection("instances").
		InsertOne(ctx, mongoData{ID: key, Value: value})

	return errors.Wrap(err, "cannot insert value")
}

func (m mongoStorage) Update(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) error {
	_, err := m.client.
		Database("atlas-broker").
		Collection("instances").
		UpdateOne(ctx, mongoData{ID: key}, mongoData{ID: key, Value: value})

	return errors.Wrap(err, "cannot update value")
}

func (m mongoStorage) Get(ctx context.Context, key string) (s *domain.GetInstanceDetailsSpec, err error) {
	result := mongoData{}
	err = m.client.
		Database("atlas-broker").
		Collection("instances").
		FindOne(ctx, mongoData{ID: key}).
		Decode(&result)

	return result.Value, errors.Wrap(err, "cannot find/decode value")
}

func (m mongoStorage) Delete(ctx context.Context, key string) error {
	_, err := m.client.
		Database("atlas-broker").
		Collection("instances").
		DeleteOne(ctx, mongoData{ID: key})

	return errors.Wrap(err, "cannot delete value")
}
