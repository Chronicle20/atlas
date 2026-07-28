package route

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the base resource path for routes
	Resource = "transports/routes"
	// RouteResource is the resource path for a specific route
	RouteResource = "transports/routes/%s"
	// RouteStateResource is the resource path for a route's state
	RouteStateResource = "transports/routes/%s/state"
	// RouteScheduleResource is the resource path for a route's schedule
	RouteScheduleResource = "transports/routes/%s/schedule"
)

// getBaseRequest returns the base URL for route requests
func getBaseRequest() string {
	return requests.RootUrl("ROUTES")
}

// inTenantUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inTenantUrl() string {
	return getBaseRequest() + Resource
}

// requestById creates a request to get a route by ID
func requestById(id string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+RouteResource, id))
}

// requestStateById creates a request to get a route's state by route ID
func requestStateById(id string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+RouteStateResource, id))
}

// requestScheduleById creates a request to get a route's schedule by route ID
func requestScheduleById(id string) requests.Request[[]TripScheduleRestModel] {
	return requests.GetRequest[[]TripScheduleRestModel](fmt.Sprintf(getBaseRequest()+RouteScheduleResource, id))
}
