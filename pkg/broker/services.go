package broker

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	atlasprivate "github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
	"github.com/pivotal-cf/brokerapi/domain"
)

// idPrefix will be prepended to service and plan IDs to ensure their uniqueness.
const idPrefix = "aosb-cluster"

// providerNames contains all the available cloud providers on which clusters
// may be provisioned. The available instance sizes for each provider are
// fetched dynamically from the Atlas API.
var (
	providerNames = []string{"AWS", "GCP", "AZURE", "TENANT"}
)

type serviceInstance struct {
	ID string `bson:"id"`
	domain.GetInstanceDetailsSpec
}

// Services generates the service catalog which will be presented to consumers of the API.
func (b *Broker) Services(ctx context.Context) ([]domain.Service, error) {
	b.logger.Info("Retrieving service catalog")
	return b.catalog.services, nil
}

func (b *Broker) buildCatalog() {
	b.catalog = newCatalog()

	svc := b.buildServiceTemplate()

	for _, p := range svc.Plans {
		b.catalog.plans[p.ID] = p
	}

	b.catalog.providers[svc.ID] = atlasprivate.Provider{Name: "template"}
	b.catalog.services = append(b.catalog.services, svc)
	b.logger.Infow("Built service", "provider", "template")
}

func (b *Broker) buildServiceTemplate() (service domain.Service) {
	return domain.Service{
		ID:                   serviceIDForProvider("template"),
		Name:                 "mongodb-atlas-template",
		Description:          "MonogoDB Atlas Plan Template Deployments",
		Bindable:             true,
		InstancesRetrievable: true,
		BindingsRetrievable:  false,
		Metadata: &domain.ServiceMetadata{
			DisplayName:         "MongoDB Atlas - Template Services",
			ImageUrl:            "https://webassets.mongodb.com/_com_assets/cms/vectors-anchor-circle-mydmar539a.svg",
			DocumentationUrl:    "https://support.mongodb.com/welcome",
			ProviderDisplayName: "MongoDB",
			LongDescription:     "Complete MongoDB Atlas deployments managed through resource templates. See https://github.com/jasonmimick/atlas-osb",
		},
		PlanUpdatable: true,
		Plans:         b.buildPlansForProviderDynamic(),
	}
}

func (b *Broker) buildPlansForProviderDynamic() []domain.ServicePlan {
	var plans []domain.ServicePlan

	templates, err := dynamicplans.FromEnv()
	if err != nil {
		b.logger.Fatalw("could not read dynamic plans from environment", "error", err)
	}

	planContext := dynamicplans.Context{
		"credentials": b.credentials,
	}

	for _, template := range templates {
		raw := new(bytes.Buffer)

		err := template.Execute(raw, planContext)
		if err != nil {
			b.logger.Errorf("cannot execute template %q: %v", template.Name(), err)
			continue
		}

		b.logger.Infof("Parsed plan: %s", raw.String())

		p := dynamicplans.Plan{}
		if err := yaml.NewDecoder(raw).Decode(&p); err != nil {
			b.logger.Errorw("cannot decode yaml template", "name", template.Name(), "error", err)
			continue
		}

		if p.Cluster == nil ||
			p.Cluster.ProviderSettings == nil {
			if p.Cluster.ProviderSettings.ProviderName == "" {
				b.logger.Errorw(
					"invalid yaml template",
					"name", template.Name(),
					"error", ".cluster.providerSettings.providerName must not be empty",
				)
				continue
			}
			if p.Cluster.ProviderSettings.InstanceSizeName == "" {
				b.logger.Errorw(
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
