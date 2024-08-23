package monster

import (
	"atlas-maps/rest"
	"atlas-maps/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	mapsResource     = "maps/"
	monstersResource = mapsResource + "%d/monsters"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestSpawnPoints(ctx context.Context, tenant tenant.Model) func(mapId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+monstersResource, mapId))
	}
}
