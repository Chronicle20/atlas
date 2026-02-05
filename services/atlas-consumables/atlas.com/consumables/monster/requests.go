package monster

import (
	"atlas-consumables/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestCreate(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	// TODO - field migration
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId()), m)
}
