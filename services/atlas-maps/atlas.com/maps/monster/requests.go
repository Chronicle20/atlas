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
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return os.Getenv("MONSTER_SERVICE_URL")
}

func requestInMap(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId))
	}
}

func requestCreate(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) requests.Request[RestModel] {
	return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) requests.Request[RestModel] {
		m := RestModel{
			Id:        "0",
			MonsterId: monsterId,
			X:         x,
			Y:         y,
			Fh:        fh,
			Team:      team,
		}
		return rest.MakePostRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId), m)
	}
}
