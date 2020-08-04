package broker

import (
	atlasprivate "github.com/mongodb/mongodb-atlas-service-broker/pkg/atlas"
	"github.com/pivotal-cf/brokerapi/domain"
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
