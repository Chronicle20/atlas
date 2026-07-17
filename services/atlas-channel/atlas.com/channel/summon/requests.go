package summon

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	summonsInMapResource = "worlds/%d/channels/%d/maps/%d/instances/%s/summons"
)

func getBaseRequest() string {
	return requests.RootUrl("SUMMONS")
}

// inMapUrl returns the list URL for the summons currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+summonsInMapResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
