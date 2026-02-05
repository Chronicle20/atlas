package reactor

import (
	"atlas-maps/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/reactors"
)

func getBaseRequest() string {
	return requests.RootUrl("REACTORS")
}

func requestInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId, channelId, mapId))
}
