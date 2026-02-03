package _map

import (
	"atlas-monster-death/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d"
	mapCharactersResource = mapResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestCharactersInField(f field.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapCharactersResource+"?instance=%s", f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
