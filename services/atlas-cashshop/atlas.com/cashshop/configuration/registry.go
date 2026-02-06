package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	mu           sync.RWMutex
	tenantConfig map[uuid.UUID]tenant.RestModel
)

func init() {
	tenantConfig = make(map[uuid.UUID]tenant.RestModel)
}

func GetTenantConfig(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (tenant.RestModel, error) {
	mu.RLock()
	if cfg, ok := tenantConfig[tenantId]; ok {
		mu.RUnlock()
		return cfg, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	if cfg, ok := tenantConfig[tenantId]; ok {
		return cfg, nil
	}

	cfg, err := RequestForTenant(tenantId)(l, ctx)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch tenant config for %s, using defaults", tenantId.String())
		cfg = tenant.RestModel{}
	}
	tenantConfig[tenantId] = cfg
	return cfg, nil
}

func GetHourlyExpirations(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) map[uint32]uint32 {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)

	result := make(map[uint32]uint32)
	for _, he := range cfg.CashShop.Commodities.HourlyExpirations {
		result[he.TemplateId] = he.Hours
	}
	return result
}
