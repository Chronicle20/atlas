package _map

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	mapResource                 = "worlds/%d/channels/%d/maps/%d"
	mapCharactersResource       = mapResource + "/characters/"
	mapInstanceResource         = mapResource + "/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInMap(f field.Model) requests.Request[[]RestModel] {
	if f.Instance() != uuid.Nil {
		return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
	}
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource, f.WorldId(), f.ChannelId(), f.MapId()))
}
