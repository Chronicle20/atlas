package configuration

import (
	"atlas-channel/configuration/tenant"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource  = "configurations"
	ByService = Resource + "/services/%s"
	ForTenant = Resource + "/tenants/%s"
)

func getBaseRequest() string {
	return requests.RootUrl("CONFIGURATIONS")
}

func requestByService(serviceId uuid.UUID) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByService, serviceId.String()))
}

func requestForTenant(tenantId uuid.UUID) requests.Request[tenant.RestModel] {
	return requests.GetRequest[tenant.RestModel](fmt.Sprintf(getBaseRequest()+ForTenant, tenantId.String()))
}
