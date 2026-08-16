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

func requestInMapById(ctx context.Context, mapId _map.Id, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+portalById, mapId, id))
}
