package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapResource                   = "worlds/%d/channels/%d/maps/%d"
	mapInstanceResource           = mapResource + "/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInField(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
