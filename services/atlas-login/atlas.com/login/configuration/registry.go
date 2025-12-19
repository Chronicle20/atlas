package configuration

import (
	"atlas-login/configuration/tenant"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
)

var once sync.Once
var serviceConfig *RestModel
var tenantConfig map[uuid.UUID]tenant.RestModel

func GetServiceConfig() (*RestModel, error) {
	if serviceConfig == nil {
		log.Fatalf("Configuration not initialized.")
	}
	return serviceConfig, nil
}

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
				c, err := requestByService(serviceId)(l, ctx)
				if err != nil {
					log.Fatalf("Could not retrieve configuration.")
				}
				serviceConfig = &c

				for _, t := range c.Tenants {
					tenantId := uuid.MustParse(t.Id)
					tc, err := requestForTenant(tenantId)(l, ctx)
					if err != nil {
						log.Fatalf("Could not retrieve tenant configuration.")
					}
					tenantConfig[tenantId] = tc
				}
			})
		}
	}
}
