package monster

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
	monstersResource    = "monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestInMap(f field.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

func requestById(uniqueId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
}
