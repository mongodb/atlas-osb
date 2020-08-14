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

package statestorage

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/mongodbrealm"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	maintenanceProjectName = "Atlas Service Broker Mainentance"
	realmAppName           = "broker-state"
)

var (
	ErrInstanceNotFound = errors.New("unable to find instance in state storage")
)

//type StateStorage interface {
//    FindOne(context.Context, string)                   (*map[string]interface{}, error)
//    InsertOne(context.Context, string, interface{})    (*map[string]interface{}, error)
//    DeleteOne(context.Context, string)                 (error)
//
//}

type RealmStateStorage struct {
	OrgID        string `json:"orgId,omitempty"`
	RealmClient  *mongodbrealm.Client
	RealmApp     *mongodbrealm.RealmApp
	RealmProject *mongodbatlas.Project
	Logger       *zap.SugaredLogger
}

func client(baseURL string, k credentials.Key) (*mongodbatlas.Client, error) {
	hc, err := digest.NewTransport(k.PublicKey, k.PrivateKey).Client()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Digest client")
	}

	return mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
}

func Get(key credentials.Key, atlasURL string, realmURL string, logger *zap.SugaredLogger) (*RealmStateStorage, error) {
	realmClient, err := mongodbrealm.New(
		context.TODO(),
		nil,
		mongodbrealm.SetBaseURL(realmURL),
		mongodbrealm.SetAPIAuth(context.TODO(), key.PublicKey, key.PrivateKey),
	)
	if err != nil {
		return nil, err
	}

	// Get or create a RealmApp for this orgID -
	// Each Organization using the broker will have 1 special
	// Atlas Group - called "Atlas Service Broker"
	//
	client, err := client(atlasURL, key)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Atlas client")
	}

	mainPrj, err := getOrCreateBrokerMaintentaceGroup(key.OrgID, client, logger)
	if err != nil {
		return nil, err
	}

	logger.Infow("Found mainteneance project", "mainPrj", mainPrj)
	realmApp, err := getOrCreateRealmAppForOrg(mainPrj.ID, realmClient, logger)
	if err != nil {
		logger.Errorw("Error getOrCreateRealmAppForOrg", "err", err)
		return nil, err
	}

	rss := &RealmStateStorage{
		OrgID:        key.OrgID,
		RealmClient:  realmClient,
		RealmApp:     realmApp,
		RealmProject: mainPrj,
		Logger:       logger,
	}
	return rss, nil
}

func getOrCreateBrokerMaintentaceGroup(orgID string, client *mongodbatlas.Client, logger *zap.SugaredLogger) (*mongodbatlas.Project, error) {
	project, _, err := client.Projects.GetOneProjectByName(context.Background(), maintenanceProjectName)
	if err != nil {
		logger.Infow("getOrCreateBrokerMaintentaceGroup", "err", err)
		prj := mongodbatlas.Project{
			Name:  maintenanceProjectName,
			OrgID: orgID,
		}

		project, _, err = client.Projects.Create(context.Background(), &prj)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create project")
		}

		logger.Infow("getOrCreateBrokerMaintentaceGroup CREATED", "project", project)
	}
	logger.Infow("getOrCreateBrokerMaintentaceGroup FOUND", "project", project)
	return project, nil
}

func getOrCreateRealmAppForOrg(groupID string, realmClient *mongodbrealm.Client, logger *zap.SugaredLogger) (*mongodbrealm.RealmApp, error) {
	app := mongodbrealm.RealmAppInput{
		Name:        realmAppName,
		ClientAppID: "atlas-osb",
		Location:    "US-VA",
		/* [US-VA, AU, US-OR, IE] */
	}

	apps, _, err := realmClient.RealmApps.List(context.Background(), groupID, nil)
	var realmApp *mongodbrealm.RealmApp

	for _, ra := range apps {
		ra := ra
		logger.Infow("Found realm app", "ra", ra)
		if ra.Name == app.Name {
			realmApp = &ra
		}
	}

	if realmApp == nil {
		logger.Infow("Error fetching maintenance realm app", "err", err)
		logger.Infow("Attempt create", "app", app)
		realmApp, _, err := realmClient.RealmApps.Create(context.Background(), groupID, &app)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create Realm app")
		}
		logger.Infow("Created realm app", "realmApp", realmApp)
		return realmApp, nil
	}

	logger.Infow("Found existing realm app", "realmApp", realmApp)
	return realmApp, nil
}

func (ss *RealmStateStorage) FindOne(ctx context.Context, key string) (*domain.GetInstanceDetailsSpec, error) {
	// Need to find the one value whose "name" = key
	values, _, err := ss.RealmClient.RealmValues.List(ctx, ss.RealmProject.ID, ss.RealmApp.ID, nil)
	if err != nil {
		// return proper InstanceNotFound, if error is realm
		if strings.Contains(err.Error(), "value not found") {
			err = ErrInstanceNotFound
		}
		return nil, err
	}

	idForKey := ""

	for _, v := range values {
		if v.Name == key {
			idForKey = v.ID
		}
	}

	val, err := ss.Get(ctx, idForKey)
	if err != nil {
		// return proper InstanceNotFound, if error is realm
		if strings.Contains(err.Error(), "value not found") {
			err = ErrInstanceNotFound
		}
		return nil, err
	}
	if val.Value == nil {
		return nil, errors.New("val.Value was nil from realm, should never happen")
	}

	spec := domain.GetInstanceDetailsSpec{}
	err = json.Unmarshal(val.Value, &spec)
	if err != nil {
		return nil, err
	}

	//spec := &domain.GetInstanceDetailsSpec{
	//    ServiceID: fmt.Sprintf("%s",val.Value["serviceID"]),
	//    PlanID: val.Value["planID"].(string),
	//    DashboardURL: val.Value["dashboardURL"].(string),
	//    Parameters: reflect.ValueOf(val.Value["parameters"]).Interface().(interface{}),
	//}

	return &spec, nil
}

func (ss *RealmStateStorage) DeleteOne(ctx context.Context, key string) error {
	_, err := ss.RealmClient.RealmValues.Delete(ctx, ss.RealmProject.ID, ss.RealmApp.ID, key)
	return err
}

func (ss *RealmStateStorage) Put(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) (*mongodbrealm.RealmValue, error) {
	vv, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	val := &mongodbrealm.RealmValue{
		Name:  key,
		Value: vv,
	}
	v, _, err := ss.RealmClient.RealmValues.Create(ctx, ss.RealmProject.ID, ss.RealmApp.ID, val)
	return v, err
}

func (ss *RealmStateStorage) Get(ctx context.Context, key string) (*mongodbrealm.RealmValue, error) {
	v, _, err := ss.RealmClient.RealmValues.Get(ctx, ss.RealmProject.ID, ss.RealmApp.ID, key)
	return v, err
}

var defaultUser = &mongodbatlas.DatabaseUser{
	Username: "admin",
	Password: "admin",
	Roles: []mongodbatlas.Role{{
		DatabaseName: "statestorage",
		RoleName:     "readWrite",
	}},
}
