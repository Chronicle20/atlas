package chair

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is missing the /instances/{instanceId} segment that
// atlas-chairs' route actually requires (world/resource.go registers
// .../maps/{mapId}/instances/{instanceId}/chairs, matching every sibling
// in-map registry service: monster, drop, reactor, summon, door). Every
// request built from the old format string 404s against the real route --
// this consumer has likely never worked. Fixed incidentally while
// converting to requests.DrainProvider (task-117), since a broken URL
// cannot be drain-tested; not a pagination-scope change.
const (
	Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/chairs"
)

func getBaseRequest() string {
	return requests.RootUrl("CHAIRS")
}

// inMapUrl returns the list URL for the chairs currently occupied in one
// map instance. It is a bare URL (not a requests.Request) because the list
// is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
