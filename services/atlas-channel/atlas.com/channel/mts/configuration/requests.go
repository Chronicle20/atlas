package configuration

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	mtsConfigResource      = "mts-configs"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's MTS
// configuration: GET /tenants/{tenantId}/configurations/mts-configs. When a
// tenant has no mts-configs row seeded the fetch misses and the registry falls
// back to DefaultConfig; seed the resource (POST .../mts-configs/seed or the
// atlas-ui config page) to drive the economic knobs per tenant.
func requestForTenant(ctx context.Context, tenantId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	url := fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId.String(), configurationsResource, mtsConfigResource)
	return requests.GetRequest[RestModel](url)
}
