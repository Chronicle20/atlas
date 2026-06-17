package configuration

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	configurationsResource = "configurations"
	mtsConfigResource      = "mts-configs"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestForTenant builds the atlas-tenants fetch for a tenant's MTS
// configuration: GET /tenants/{tenantId}/configurations/mts-configs. The
// resource is not implemented in atlas-tenants until Phase 8, so this fetch
// misses at runtime today and the registry falls back to defaults.
func requestForTenant(tenantId uuid.UUID) requests.Request[RestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId.String(), configurationsResource, mtsConfigResource)
	return requests.GetRequest[RestModel](url)
}
