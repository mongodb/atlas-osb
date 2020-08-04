package broker

import (
	"context"

	"github.com/pivotal-cf/brokerapi/domain"
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
	return err
}

func (m mongoStorage) Update(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) error {
	_, err := m.client.
		Database("atlas-broker").
		Collection("instances").
		UpdateOne(ctx, mongoData{ID: key}, mongoData{ID: key, Value: value})
	return err
}

func (m mongoStorage) Get(ctx context.Context, key string) (s *domain.GetInstanceDetailsSpec, err error) {
	result := mongoData{}
	err = m.client.
		Database("atlas-broker").
		Collection("instances").
		FindOne(ctx, mongoData{ID: key}).
		Decode(&result)

	return result.Value, err
}

func (m mongoStorage) Delete(ctx context.Context, key string) error {
	_, err := m.client.
		Database("atlas-broker").
		Collection("instances").
		DeleteOne(ctx, mongoData{ID: key})
	return err
}
