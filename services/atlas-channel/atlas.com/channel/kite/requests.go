package kite

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource carries the /instances/{instanceId} segment from day one — the
// chalkboard sibling shipped without it and 404'd against the real route for
// its entire life (chalkboard/requests.go:9-17).
const Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/kites"

func getBaseRequest() string {
	return requests.RootUrl("KITES")
}

// inMapUrl returns the list URL for the kites currently displayed in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// paginated server-side and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
