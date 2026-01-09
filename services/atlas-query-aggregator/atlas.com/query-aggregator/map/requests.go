package _map

import (
	"atlas-query-aggregator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS") + "/worlds"
}

func requestCharactersInMap(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](
		fmt.Sprintf(getBaseRequest()+"/%d/channels/%d/maps/%d/characters", worldId, channelId, mapId),
	)
}
