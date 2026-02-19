package configuration

import (
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
