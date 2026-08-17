package portal

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	portalsInMap  = "data/maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
	portalById    = portalsInMap + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// inMapUrl is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func inMapUrl(ctx context.Context, mapId _map.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+portalsInMap, mapId), nil
}

func requestInMapByName(ctx context.Context, mapId _map.Id, name string) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+portalsByName, mapId, name))
}

func requestInMapById(ctx context.Context, mapId _map.Id, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+portalById, mapId, id))
}
