package monster

import (
	"atlas-maps/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId))
}

func requestCreate(worldId world.Id, channelId channel.Id, mapId _map.Id, monsterId uint32, x int16, y int16, fh uint16, team int32) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId), m)
}
