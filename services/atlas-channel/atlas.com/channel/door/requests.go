package door

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	resourceById    = "doors/%s"
	resourceInField = "worlds/%d/channels/%d/maps/%d/instances/%s/doors"
	resourceByOwner = "characters/%d/doors"
)

func getBaseRequest() string {
	return requests.RootUrl("DOORS")
}

func requestById(id string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+resourceById, id))
}

// inFieldUrl returns the list URL for the doors currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inFieldUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+resourceInField, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// byOwnerUrl returns the list URL for a character's doors. Bare URL for the
// same reason as inFieldUrl.
func byOwnerUrl(ownerCharacterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+resourceByOwner, ownerCharacterId)
}
