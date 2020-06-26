package broker

import (
	"errors"
	"fmt"
	"net/http"

	atlasprivate "github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
)

type catalog struct {
	services  []domain.Service
	providers map[string]atlasprivate.Provider
	plans     map[string]domain.ServicePlan
}

func newCatalog() *catalog {
	return &catalog{
		services:  []domain.Service{},
		providers: map[string]atlasprivate.Provider{},
		plans:     map[string]domain.ServicePlan{},
	}
}

func (c catalog) findInstanceSizeByPlanID(provider *atlasprivate.Provider, planID string) (*atlasprivate.InstanceSize, error) {
	p, found := c.plans[planID]
	if !found {
		return nil, fmt.Errorf("plan ID %q not found in catalog", planID)
	}

	szi, found := p.Metadata.AdditionalMetadata["instanceSize"]
	if !found {
		return nil, fmt.Errorf("instance size not found in metadata for plan %q", planID)
	}

	sz, ok := szi.(atlasprivate.InstanceSize)
	if !ok {
		return nil, fmt.Errorf("incorrect metadata type: expected atlasprivate.InstanceSize, found %T", szi)
	}

	return &sz, nil
}

func (c *catalog) findGroupIDByPlanID(planID string) (string, error) {
	p, found := c.plans[planID]
	if !found {
		return "", fmt.Errorf("plan ID %q not found in catalog", planID)
	}

	gidi, found := p.Metadata.AdditionalMetadata["groupID"]
	if !found {
		return "", fmt.Errorf("group ID not found in metadata for plan %q", planID)
	}

	gid, ok := gidi.(string)
	if !ok {
		return "", fmt.Errorf("incorrect metadata type: expected string, found %T", gidi)
	}

	return gid, nil
}

func (c *catalog) findProviderByServiceID(serviceID string) (*atlasprivate.Provider, error) {
	p, found := c.providers[serviceID]
	if !found {
		return nil, apiresponses.NewFailureResponse(errors.New("Invalid service ID"), http.StatusBadRequest, "invalid-service-id")
	}
	return &p, nil
}

// applyWhitelist filters a given service, returning the service with only the
// whitelisted plans.
func (c *catalog) applyWhitelist(plans []domain.ServicePlan, whitelist []string) []domain.ServicePlan {
	whitelistedPlans := []domain.ServicePlan{}

	for _, plan := range plans {
		for _, name := range whitelist {
			if plan.Name == name {
				whitelistedPlans = append(plans, plan)
				break
			}
		}
	}

	return whitelistedPlans
}
