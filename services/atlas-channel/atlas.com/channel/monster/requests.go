package monster

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"os"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
	monstersResource    = "monsters/%d"
)

func getBaseRequest() string {
	return os.Getenv("MONSTER_SERVICE_URL")
}

func requestInMap(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId))
	}
}

func requestById(ctx context.Context, tenant tenant.Model) func(uniqueId uint32) requests.Request[RestModel] {
	return func(uniqueId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
	}
}
