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

func requestInMap(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

// requestInMapRect issues the atlas-monsters rectangle query for AoE skill
// targeting (e.g., Priest Doom). Bounds are inclusive; limit == 0 means "no cap".
func requestInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersRectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String(), x1, y1, x2, y2, limit))
}

func requestById(uniqueId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
}
