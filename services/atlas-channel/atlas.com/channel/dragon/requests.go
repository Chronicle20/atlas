package dragon

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const dragonsInMapResource = "worlds/%d/channels/%d/maps/%d/instances/%s/dragons"

func getBaseRequest() string {
	return requests.RootUrl("DRAGONS")
}

// inMapUrl returns the list URL for the dragons currently in one map instance.
// It is a bare URL (not a requests.Request) because the list is paginated
// server-side and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] params.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+dragonsInMapResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
