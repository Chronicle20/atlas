package config

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	instanceRoutesResource = "instance-routes"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

func requestInstanceRoutes(tenantId string) requests.Request[[]InstanceRouteRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, instanceRoutesResource)
	return requests.GetRequest[[]InstanceRouteRestModel](url)
}
