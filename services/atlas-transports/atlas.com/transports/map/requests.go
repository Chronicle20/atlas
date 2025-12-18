package _map

import (
	"atlas-transports/rest"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d"
	mapCharactersResource = mapResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource, worldId, channelId, mapId))
}
