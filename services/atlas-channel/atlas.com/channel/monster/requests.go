package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapMonstersResource     = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
	mapMonstersRectResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters/in-rect?x1=%d&y1=%d&x2=%d&y2=%d&limit=%d"
	monstersResource        = "monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

// inMapUrl returns the list URL for the monsters currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// inMapRectUrl returns the list URL for the atlas-monsters rectangle query
// used for AoE skill targeting (e.g., Priest Doom). Bounds are inclusive;
// limit == 0 means "no cap". Bare URL for the same reason as inMapUrl --
// atlas-monsters preserves its ascending-distance-from-center order across
// pages, so draining is still safe (page order is meaningful, not
// re-sorted).
func inMapRectUrl(f field.Model, x1, y1, x2, y2 int16, limit uint32) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersRectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String(), x1, y1, x2, y2, limit)
}

func requestById(uniqueId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
}
