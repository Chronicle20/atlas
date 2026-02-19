package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapCharactersResource = mapResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(field field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()))
}
