package chalkboard

import (
	"atlas-channel/rest"
	"fmt"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/chalkboards"
)

func getBaseRequest() string {
	return requests.RootUrl("CHALKBOARDS")
}

func requestInMap(m _map.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, m.WorldId(), m.ChannelId(), m.MapId()))
}
