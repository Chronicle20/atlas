package monster

import (
	"atlas-consumables/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestCreate(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) requests.Request[RestModel] {
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
