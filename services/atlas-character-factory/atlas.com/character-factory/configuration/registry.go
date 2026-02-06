package configuration

import (
	"atlas-character-factory/configuration/tenant"
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var once sync.Once
var tenantConfig map[uuid.UUID]tenant.RestModel

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	var val tenant.RestModel
	var ok bool
	if val, ok = tenantConfig[tenantId]; !ok {
		log.Fatalf("tenant not configured")
	}
	return val, nil
}

func Init(l logrus.FieldLogger) func(ctx context.Context) func(serviceId uuid.UUID) {
	return func(ctx context.Context) func(serviceId uuid.UUID) {
		return func(serviceId uuid.UUID) {
			once.Do(func() {
				tenantConfig = make(map[uuid.UUID]tenant.RestModel)
				tcs, err := requestAllTenants()(l, ctx)
				if err != nil {
					log.Fatalf("Could not retrieve tenant configuration.")
				}

				for _, tc := range tcs {
					tenantConfig[uuid.MustParse(tc.Id)] = tc
				}
			})
		}
	}
}
