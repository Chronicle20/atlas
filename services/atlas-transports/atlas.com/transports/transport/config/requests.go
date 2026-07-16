package config

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	routesResource         = "routes"
	vesselsResource        = "vessels"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// routesUrl and vesselsUrl are bare URLs (not requests.Request) because
// both lists are now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func routesUrl(tenantId string) string {
	return fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, routesResource)
}

func vesselsUrl(tenantId string) string {
	return fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, vesselsResource)
}
