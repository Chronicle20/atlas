package configuration

import (
	"atlas-world/configuration/tenant"
	"atlas-world/rate"
	"context"
	"log"
	"sync"

	"github.com/Chronicle20/atlas-constants/world"
	tenant2 "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
					l.WithError(err).Fatalf("Could not retrieve tenant configuration.")
				}

				for _, tc := range tcs {
					tenantId := uuid.MustParse(tc.Id)
					tenantConfig[tenantId] = tc

					// Initialize world rates from configuration
					initializeRatesFromConfig(l, tenantId, tc)
				}
			})
		}
	}
}

// initializeRatesFromConfig initializes the rate registry with rates from configuration
func initializeRatesFromConfig(l logrus.FieldLogger, tenantId uuid.UUID, tc tenant.RestModel) {
	t, err := tenant2.Create(tenantId, tc.Region, tc.MajorVersion, tc.MinorVersion)
	if err != nil {
		l.WithError(err).Errorf("Unable to create tenant model for rate initialization.")
		return
	}

	ctx := tenant2.WithContext(context.Background(), t)
	for worldId, wc := range tc.Worlds {
		rates := rate.NewModel()
		rates = rates.WithRate(rate.TypeExp, wc.GetExpRate())
		rates = rates.WithRate(rate.TypeMeso, wc.GetMesoRate())
		rates = rates.WithRate(rate.TypeItemDrop, wc.GetItemDropRate())
		rates = rates.WithRate(rate.TypeQuestExp, wc.GetQuestExpRate())

		rate.GetRegistry().InitWorldRates(ctx, world.Id(worldId), rates)
		l.Infof("Initialized world [%d] rates from config: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
			worldId, wc.GetExpRate(), wc.GetMesoRate(), wc.GetItemDropRate(), wc.GetQuestExpRate())
	}
}
