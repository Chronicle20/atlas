package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapResource                   = "worlds/%d/channels/%d/maps/%d"
	mapInstanceResource           = mapResource + "/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
	characterLocationResource     = "characters/%d/location"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

// charactersInFieldUrl returns the list URL for the characters currently in
// one map instance. It is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func charactersInFieldUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func requestCharacterLocation(characterId uint32) requests.Request[LocationRestModel] {
	return requests.GetRequest[LocationRestModel](fmt.Sprintf(getBaseRequest()+characterLocationResource, characterId))
}
