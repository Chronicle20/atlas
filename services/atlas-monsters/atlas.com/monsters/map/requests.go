package _map

import (
	"atlas-monsters/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d"
	mapCharactersResource = mapResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource, worldId, channelId, mapId))
}
