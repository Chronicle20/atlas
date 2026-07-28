package reactor

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/reactors"
)

func getBaseRequest() string {
	return requests.RootUrl("REACTORS")
}

// inMapUrl returns the list URL for the reactors currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(field field.Model) string {
	return fmt.Sprintf(getBaseRequest()+Resource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance())
}
