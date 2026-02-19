package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource  = "configurations"
	ForTenant = Resource + "/tenants/%s"
)

func getBaseRequest() string {
	return requests.RootUrl("CONFIGURATIONS")
}

func RequestForTenant(tenantId uuid.UUID) requests.Request[tenant.RestModel] {
	return requests.GetRequest[tenant.RestModel](fmt.Sprintf(getBaseRequest()+ForTenant, tenantId.String()))
}
