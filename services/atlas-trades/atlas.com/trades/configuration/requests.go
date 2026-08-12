package configuration

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	tradeConfigResource    = "trade-configs"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's trade
// configuration: GET /tenants/{tenantId}/configurations/trade-configs. When a
// tenant has no trade-configs row seeded the fetch misses and the registry
// falls back to DefaultConfig (FR-9.2); seed the resource
// (POST .../trade-configs/seed) to drive the knobs per tenant.
func requestForTenant(tenantId uuid.UUID) requests.Request[RestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId.String(), configurationsResource, tradeConfigResource)
	return requests.GetRequest[RestModel](url)
}
