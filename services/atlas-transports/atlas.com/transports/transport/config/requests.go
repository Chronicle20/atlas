package config

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	routesResource         = "routes"
	vesselsResource        = "vessels"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestRoutes creates a request for routes for a tenant
func requestRoutes(tenantId string) requests.Request[[]RouteRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, routesResource)
	return requests.GetRequest[[]RouteRestModel](url)
}

// requestVessels creates a request for vessels for a tenant
func requestVessels(tenantId string) requests.Request[[]VesselRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, vesselsResource)
	return requests.GetRequest[[]VesselRestModel](url)
}
