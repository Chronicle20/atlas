package configuration

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	mtsConfigResource      = "mts-configs"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's MTS
// configuration: GET /tenants/{tenantId}/configurations/mts-configs. When a
// tenant has no mts-configs row seeded the fetch misses and the registry falls
// back to DefaultConfig; seed the resource (POST .../mts-configs/seed or the
// atlas-ui config page) to drive the economic knobs per tenant.
func requestForTenant(tenantId uuid.UUID) requests.Request[RestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId.String(), configurationsResource, mtsConfigResource)
	return requests.GetRequest[RestModel](url)
}
