package _map

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapCharactersResource = mapResource + "/characters/"
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
	return fmt.Sprintf(getBaseRequest()+mapCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance())
}
