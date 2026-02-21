package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapInstanceResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
	mapCharactersResource         = "worlds/%d/channels/%d/maps/%d/characters"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

func requestCharactersInMapAllInstances(worldId world.Id, channelId channel.Id, mapId _map.Id) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource, worldId, channelId, mapId))
}
