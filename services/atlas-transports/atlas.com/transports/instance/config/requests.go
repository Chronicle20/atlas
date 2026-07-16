package config

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	instanceRoutesResource = "instance-routes"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// instanceRoutesUrl is a bare URL (not a requests.Request) because the list
// is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func instanceRoutesUrl(tenantId string) string {
	return fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, instanceRoutesResource)
}
