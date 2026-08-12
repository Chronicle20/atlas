package configuration

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	kiteConfigResource     = "kite-configs"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's kite
// configuration: GET /tenants/{tenantId}/configurations/kite-configs. When a
// tenant has no kite-configs row seeded the fetch misses and the registry
// falls back to DefaultConfig.
func requestForTenant(tenantId uuid.UUID) requests.Request[RestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId.String(), configurationsResource, kiteConfigResource)
	return requests.GetRequest[RestModel](url)
}
