package events

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// visualsInMapResource is atlas-events' Task 16 map-entry projection route:
// event/occurrence/resource.go:34 registers
// "/events/worlds/{worldId}/channels/{channelId}/maps/{mapId}/visuals".
const visualsInMapResource = "worlds/%d/channels/%d/maps/%d/visuals"

func getBaseRequest() string {
	return requests.RootUrl("EVENTS")
}

func requestVisualsInMap(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+visualsInMapResource, f.WorldId(), f.ChannelId(), f.MapId()))
}
