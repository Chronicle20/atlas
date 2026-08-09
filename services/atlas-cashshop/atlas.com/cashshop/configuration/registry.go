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

// DefaultSurpriseBoxTemplateId is the stock Cash Shop Surprise box. It is
// the fallback, not a constant the open path compares against directly:
// GetSurpriseBoxTemplateIds is the only reader, and a tenant may override
// or extend the list.
const DefaultSurpriseBoxTemplateId = uint32(5222000)

// GetSurpriseBoxTemplateIds returns the cash item template ids that open as
// a Cash Shop Surprise box for this tenant. GetTenantConfig returns a zero
// RestModel when the fetch fails, so an empty list means "unconfigured" and
// falls back to the stock box rather than disabling the feature.
func GetSurpriseBoxTemplateIds(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []uint32 {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)
	if len(cfg.CashShop.Surprise.BoxTemplateIds) == 0 {
		return []uint32{DefaultSurpriseBoxTemplateId}
	}
	return cfg.CashShop.Surprise.BoxTemplateIds
}
