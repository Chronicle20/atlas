package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"atlas-cashshop/rest"
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
	return rest.MakeGetRequest[tenant.RestModel](fmt.Sprintf(getBaseRequest()+ForTenant, tenantId.String()))
}
