package configuration

import (
	"atlas-login/configuration/tenant"
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var once sync.Once
var configMu sync.RWMutex
var serviceConfig *RestModel
var tenantConfig map[uuid.UUID]tenant.RestModel

func GetServiceConfig() (*RestModel, error) {
	configMu.RLock()
	defer configMu.RUnlock()
	if serviceConfig == nil {
		log.Fatalf("Configuration not initialized.")
	}
	return serviceConfig, nil
}

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	configMu.RLock()
	defer configMu.RUnlock()
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

// PublishSnapshot replaces the package-level service+tenant config with
// the snapshot taken from the kafka-backed projection. Called by main.go
// after CaughtUp fires (and again from the projection apply loop on each
// observed change) so legacy callers of GetServiceConfig / GetTenantConfig
// (handlers, the session timeout task, the account-session kafka
// consumer) see the same data the listener registry was built from. Both
// args are taken by value-copy so the caller's projection State can
// mutate independently after the call.
func PublishSnapshot(svc *RestModel, tenants map[uuid.UUID]tenant.RestModel) {
	configMu.Lock()
	defer configMu.Unlock()
	if svc != nil {
		c := *svc
		serviceConfig = &c
	} else {
		serviceConfig = nil
	}
	next := make(map[uuid.UUID]tenant.RestModel, len(tenants))
	for k, v := range tenants {
		next[k] = v
	}
	tenantConfig = next
}
