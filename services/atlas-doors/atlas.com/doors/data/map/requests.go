package map_

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	getMap        = "data/maps/%d"
	getMapPortals = "data/maps/%d/portals"
)

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

// requestMap fetches a map with portals included via ?include=portals.
func requestMap(ctx context.Context, mapId _map.Id) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	url := fmt.Sprintf(root+getMap+"?include=portals", mapId)
	return requests.GetRequest[RestModel](url)
}

// portalsUrl is a bare URL (not a requests.Request) for the /portals
// sub-resource: the list is now paginated server-side (task-117) and
// consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func portalsUrl(ctx context.Context, mapId _map.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+getMapPortals, mapId), nil
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrlFor("DATA") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
