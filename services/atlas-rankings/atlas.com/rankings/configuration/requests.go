package configuration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// DefaultRecomputeInterval applies when a tenant has no rankings
// configuration (or the read fails) — FR-4.
const DefaultRecomputeInterval = 5 * time.Minute

const byTenant = "tenants/%s/configurations/rankings"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

func requestByTenantId(ctx context.Context, tenantId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+byTenant, tenantId))
}

// GetRecomputeInterval resolves the tenant's recompute cadence. Missing
// config (404) is the expected unconfigured state; any other error is
// logged. Both fall back to the default so one tenant's config problem
// never stalls its recompute entirely.
func GetRecomputeInterval(l logrus.FieldLogger, ctx context.Context) func(tenantId uuid.UUID) time.Duration {
	return func(tenantId uuid.UUID) time.Duration {
		rm, err := requestByTenantId(ctx, tenantId)(l, ctx)
		if err != nil {
			if !errors.Is(err, requests.ErrNotFound) {
				l.WithError(err).Warnf("Unable to read rankings configuration for tenant [%s]; using default interval.", tenantId)
			}
			return DefaultRecomputeInterval
		}
		if rm.RecomputeIntervalMinutes == 0 {
			return DefaultRecomputeInterval
		}
		return time.Duration(rm.RecomputeIntervalMinutes) * time.Minute
	}
}
