package route

import (
	"context"
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
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "ROUTES")
}

// inTenantUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inTenantUrl(ctx context.Context) string {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + Resource, nil
}

// requestById creates a request to get a route by ID
func requestById(ctx context.Context, id string) requests.Request[RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+RouteResource, id))
}

// requestStateById creates a request to get a route's state by route ID
func requestStateById(ctx context.Context, id string) requests.Request[RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+RouteStateResource, id))
}

// requestScheduleById creates a request to get a route's schedule by route ID
func requestScheduleById(ctx context.Context, id string) requests.Request[[]TripScheduleRestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]TripScheduleRestModel](err)
	}
	return requests.GetRequest[[]TripScheduleRestModel](fmt.Sprintf(root+RouteScheduleResource, id))
}
