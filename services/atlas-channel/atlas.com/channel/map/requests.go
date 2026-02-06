package _map

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapInstanceResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(f field.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
