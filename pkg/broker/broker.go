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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/goccy/go-yaml"
	"github.com/gorilla/mux"
	"github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/mongodb/atlas-osb/pkg/broker/statestorage"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Ensure broker adheres to the ServiceBroker interface.
var _ domain.ServiceBroker = new(Broker)

// Broker is responsible for translating OSB calls to Atlas API calls.
// Implements the domain.ServiceBroker interface making it easy to spin up
// an API server.
type Broker struct {
	logger      *zap.SugaredLogger
	credentials *credentials.Credentials
	cfg         Config
	catalog     *catalog
	userAgent   string
}

type Config struct {
	AtlasURL            string
	RealmURL            string
	Host                string
	Port                uint16
	CertPath            string
	KeyPath             string
	ServiceName         string
	ServiceDisplayName  string
	ServiceDesc         string
	ServiceTags         string
	ImageURL            string
	DocumentationURL    string
	ProviderDisplayName string
	LongDescription     string
}

// New creates a new Broker with a logger.
func New(
	logger *zap.SugaredLogger,
	credentials *credentials.Credentials,
	cfg Config,
	userAgent string,
) *Broker {
	b := &Broker{
		logger:      logger,
		credentials: credentials,
		cfg:         cfg,
		userAgent:   userAgent,
	}

	b.buildCatalog()
	return b
}

func (b *Broker) funcLogger() *zap.SugaredLogger {
	pc := []uintptr{0}
	runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc)
	f, _ := frames.Next()
	split := strings.Split(f.Function, ".")
	return b.logger.With("func", split[len(split)-1])
}

func (b *Broker) parsePlan(ctx dynamicplans.Context, planID string) (dp *dynamicplans.Plan, err error) {
	logger := b.funcLogger()
	sp, ok := b.catalog.plans[planID]
	if !ok {
		err = fmt.Errorf("plan ID %q not found in catalog", planID)
		return
	}

	tpl, ok := sp.Metadata.AdditionalMetadata["template"].(dynamicplans.TemplateContainer)
	if !ok {
		err = errors.New("plan ID %q does not contain a valid plan template")
		return
	}

	raw := new(bytes.Buffer)
	err = tpl.Execute(raw, ctx.With("credentials", b.credentials))
	if err != nil {
		return
	}

	dp = &dynamicplans.Plan{}
	if err = yaml.NewDecoder(raw).Decode(dp); err != nil {
		return
	}

	logger.Infow("Parsed plan", "plan", dp.SafeCopy())

	// Attempt to merge in any other values as plan instance data
	pb, _ := json.Marshal(ctx)
	logger.Infow("Found plan instance data to merge", "pb", pb)
	err = json.Unmarshal(pb, &dp)
	if err != nil {
		logger.Errorw("Error trying to merge in planContext as plan instance", "err", err)
	} else {
		logger.Infow("Merged final plan instance:", "plan", dp.SafeCopy())
	}

	return dp, nil
}

func (b *Broker) getInstancePlan(ctx context.Context, instanceID string) (*dynamicplans.Plan, error) {
	i, err := b.getInstance(ctx, instanceID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch instance")
	}

	params, ok := i.Parameters.(string)
	if !ok {
		return nil, fmt.Errorf("instance metadata has the wrong type %T", i.Parameters)
	}

	plan, err := decodePlan(params)
	return &plan, err
}

func (b *Broker) getPlan(ctx context.Context, instanceID string, planID string, planCtx dynamicplans.Context) (dp *dynamicplans.Plan, err error) {
	dp, err = b.getInstancePlan(ctx, instanceID)
	if err == nil {
		return
	}

	// planCtx == nil means the instance should exist
	if planCtx == nil {
		err = errors.Wrapf(err, "cannot find plan for instance %q", instanceID)
		return
	}

	// new instance: get from plan
	dp, err = b.parsePlan(planCtx, planID)
	if err != nil {
		return
	}

	if dp.Project == nil {
		err = fmt.Errorf("missing Project in plan definition")
		return
	}

	return
}

func (b *Broker) getClient(ctx context.Context, instanceID string, planID string, planCtx dynamicplans.Context) (client *mongodbatlas.Client, dp *dynamicplans.Plan, err error) {
	dp, err = b.getPlan(ctx, instanceID, planID, planCtx)
	if err != nil {
		return
	}

	key := credentials.APIKey{}

	switch {
	case dp.APIKey != nil:
		key = *dp.APIKey
		dp.Project.OrgID = dp.APIKey.OrgID

	case dp.Project.OrgID != "":
		key, err = b.credentials.ByOrg(dp.Project.OrgID)
		if err != nil {
			return
		}

	default:
		err = errors.New("template must contain either APIKey or Project.OrgID")
		return
	}

	hc, err := digest.NewTransport(key.PublicKey, key.PrivateKey).Client()
	if err != nil {
		err = errors.Wrap(err, "cannot create Digest client")
		return
	}

	client, err = mongodbatlas.New(hc, mongodbatlas.SetBaseURL(b.cfg.AtlasURL))
	if err != nil {
		err = errors.Wrap(err, "cannot create Atlas client")
		return
	}

	// try to merge existing project into plan, don't error out if not found
	var existing *mongodbatlas.Project
	existing, _, err = client.Projects.GetOneProjectByName(ctx, dp.Project.Name)
	if err == nil {
		dp.Project = existing
		return
	}

	err = nil
	return
}

func (b *Broker) getState(orgID string) (*statestorage.RealmStateStorage, error) {
	key, err := b.credentials.ByOrg(orgID)
	if err != nil {
		return nil, err
	}

	return statestorage.Get(key, b.cfg.AtlasURL, b.cfg.RealmURL, b.logger)
}

func (b *Broker) AuthMiddleware() mux.MiddlewareFunc {
	if b.credentials != nil {
		return authMiddleware(*b.credentials.Broker)
	}

	return simpleAuthMiddleware(b.cfg.AtlasURL)
}

func (b *Broker) GetDashboardURL(groupID, clusterName string) string {
	apiURL, err := url.Parse(b.cfg.AtlasURL)
	if err != nil {
		return err.Error()
	}
	apiURL.Path = fmt.Sprintf("/v2/%s", groupID)
	return apiURL.String() + fmt.Sprintf("#clusters/detail/%s", clusterName)
}

func encodePlan(v dynamicplans.Plan) (string, error) {
	b := new(bytes.Buffer)
	b64 := base64.NewEncoder(base64.StdEncoding, b)
	err := json.NewEncoder(b64).Encode(v)
	if err != nil {
		return "", err
	}

	err = b64.Close()
	return b.String(), err
}

func decodePlan(enc string) (dynamicplans.Plan, error) {
	b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(enc))
	dp := dynamicplans.Plan{}
	err := json.NewDecoder(b64).Decode(&dp)
	return dp, err
}
