package portal

import (
	"atlas-channel/rest"
	"atlas-channel/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	portalsInMap  = "maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestInMapByName(ctx context.Context, tenant tenant.Model) func(mapId uint32, name string) requests.Request[[]RestModel] {
	return func(mapId uint32, name string) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
	}
}
