package configuration

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	imprintConfigResource  = "imprint-configs"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's imprint
// configuration: GET /tenants/{tenantId}/configurations/imprint-configs. When
// a tenant has no imprint-configs row seeded the fetch misses and the
// registry falls back to DefaultConfig (168h); seed the resource
// (POST .../imprint-configs/seed) to drive the expiry per tenant.
func requestForTenant(tenantId uuid.UUID) requests.Request[RestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId.String(), configurationsResource, imprintConfigResource)
	return requests.GetRequest[RestModel](url)
}
