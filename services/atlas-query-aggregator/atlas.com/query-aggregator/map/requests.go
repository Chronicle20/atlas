package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapInstanceResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

// charactersInMapUrl returns the list URL for the characters currently in
// one map instance. It is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func charactersInMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
