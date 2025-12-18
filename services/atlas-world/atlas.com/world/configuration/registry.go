package configuration

import (
	"atlas-world/configuration/tenant"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
)

var once sync.Once
var tenantConfig map[uuid.UUID]tenant.RestModel

func GetTenantConfigs() map[uuid.UUID]tenant.RestModel {
	if tenantConfig == nil || len(tenantConfig) == 0 {
		log.Fatalf("tenant not configured")
	}
	return tenantConfig
}

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
