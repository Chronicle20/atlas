package monster

import (
	"atlas-maps/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestInMap(field field.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()))
}

func requestCreate(field field.Model, monsterId uint32, x int16, y int16, fh uint16, team int32) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()), m)
}
