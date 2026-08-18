package configuration

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	kiteConfigResource     = "kite-configs"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's kite
// configuration: GET /tenants/{tenantId}/configurations/kite-configs. When a
// tenant has no kite-configs row seeded the fetch misses and the registry
// falls back to DefaultConfig.
func requestForTenant(ctx context.Context, tenantId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	url := fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId.String(), configurationsResource, kiteConfigResource)
	return requests.GetRequest[RestModel](url)
}
