package monster

import (
	"atlas-channel/rest"
	"fmt"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
	monstersResource    = "monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestInMap(m _map.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, m.WorldId(), m.ChannelId(), m.MapId()))
}

func requestById(uniqueId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
}
