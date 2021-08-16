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
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
)

// idPrefix will be prepended to service and plan IDs to ensure their uniqueness.
const idPrefix = "aosb-cluster"

// Services generates the service catalog which will be presented to consumers of the API.
func (b *Broker) Services(ctx context.Context) ([]domain.Service, error) {
	logger := b.funcLogger()
	logger.Info("Retrieving service catalog")

	return b.catalog.services, nil
}

func (b *Broker) buildCatalog() {
	logger := b.funcLogger()
	b.catalog = newCatalog()

	svc := b.buildServiceTemplate()

	for _, p := range svc.Plans {
		b.catalog.plans[p.ID] = p
	}

	b.catalog.providers[svc.ID] = Provider{Name: "template"}
	b.catalog.services = append(b.catalog.services, svc)
	logger.Infow("Built service", "provider", "template")
}

func (b *Broker) buildServiceTemplate() (service domain.Service) {
	return domain.Service{
		ID:                   serviceIDForProvider("template"),
		Name:                 b.cfg.ServiceName,
		Description:          b.cfg.ServiceDesc,
		Tags:                 strings.Split(b.cfg.ServiceTags, ","),
		Bindable:             true,
		InstancesRetrievable: true,
		BindingsRetrievable:  false,
		Metadata: &domain.ServiceMetadata{
			DisplayName:         fmt.Sprintf("MongoDB Atlas - %s", b.cfg.ServiceDisplayName),
			ImageUrl:            b.cfg.ImageURL,
			DocumentationUrl:    b.cfg.DocumentationURL,
			ProviderDisplayName: b.cfg.ProviderDisplayName,
			LongDescription:     b.cfg.LongDescription,
		},
		PlanUpdatable: true,
		Plans:         b.buildPlansForProviderDynamic(),
	}
}

func (b *Broker) buildPlansForProviderDynamic() []domain.ServicePlan {
	logger := b.funcLogger()

	templates, err := dynamicplans.FromEnv()
	if err != nil {
		logger.Fatalw("could not read dynamic plans from environment", "error", err)
	}

	planContext := dynamicplans.Context{
		"credentials": b.credentials,
	}

	plans := make([]domain.ServicePlan, 0, len(templates))

	for _, template := range templates {
		raw := new(bytes.Buffer)

		err := template.Execute(raw, planContext)
		if err != nil {
			logger.Errorf("cannot execute template %q: %v", template.Name(), err)

			continue
		}

		p := dynamicplans.Plan{}
		if err := yaml.NewDecoder(raw).Decode(&p); err != nil {
			logger.Errorw("cannot decode yaml template", "name", template.Name(), "error", err)

			continue
		}

		logger.Infof("Parsed plan: %s", p.SafeCopy())

		if p.Cluster == nil ||
			p.Cluster.ProviderSettings == nil {
			if p.Cluster.ProviderSettings.ProviderName == "" {
				logger.Errorw(
					"invalid yaml template",
					"name", template.Name(),
					"error", ".cluster.providerSettings.providerName must not be empty",
				)

				continue
			}
			if p.Cluster.ProviderSettings.InstanceSizeName == "" {
				logger.Errorw(
					"invalid yaml template",
					"name", template.Name(),
					"error", ".cluster.providerSettings.instanceSizeName must not be empty",
				)

				continue
			}
		}

		plan := domain.ServicePlan{
			ID:          planIDForDynamicPlan("template", p.Name),
			Name:        p.Name,
			Description: p.Description,
			Free:        p.Free,
			Metadata: &domain.ServicePlanMetadata{
				DisplayName: p.Name,
				Bullets:     []string{p.Description},
				AdditionalMetadata: map[string]interface{}{
					"template":     dynamicplans.TemplateContainer{Template: template},
					"instanceSize": p.Cluster.ProviderSettings.InstanceSizeName,
				},
			},
		}
		plans = append(plans, plan)

		continue
	}

	return plans
}

// serviceIDForProvider will generate a globally unique ID for a provider.
func serviceIDForProvider(providerName string) string {
	return fmt.Sprintf("%s-service-%s", idPrefix, strings.ToLower(providerName))
}

func planIDForDynamicPlan(providerName string, planName string) string {
	return fmt.Sprintf("%s-plan-%s-%s", idPrefix, strings.ToLower(providerName), strings.ToLower(planName))
}
