package _map

import (
	"atlas-monsters/rest"
	"atlas-monsters/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d"
	mapCharactersResource = mapResource + "/characters/"
)

func getBaseRequest() string {
	return os.Getenv("MAP_SERVICE_URL")
}

func requestCharactersInMap(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+mapCharactersResource, worldId, channelId, mapId))
	}
}
