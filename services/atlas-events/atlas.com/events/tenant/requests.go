package tenant

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	tenantsResource = "tenants"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// allTenantsUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func allTenantsUrl() string {
	return getBaseRequest() + tenantsResource
}
