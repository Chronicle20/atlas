package portal

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	portalsInMap  = "maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
	portalById    = portalsInMap + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestInMapByName(ctx context.Context, tenant tenant.Model) func(mapId uint32, name string) requests.Request[[]RestModel] {
	return func(mapId uint32, name string) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
	}
}

func requestInMapById(ctx context.Context, tenant tenant.Model) func(mapId uint32, id uint32) requests.Request[RestModel] {
	return func(mapId uint32, id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+portalById, mapId, id))
	}
}
