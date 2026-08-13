package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
	// The result cap is spelled `max`, NOT `limit`. Any request carrying a
	// `limit` query param is rejected with 400 by paginate.ParseParams
	// (libs/atlas-rest/server/paginate/params.go) -- the repo-wide ban that
	// makes page[number]/page[size] the only paging vocabulary (task-117).
	// This URL is drained through requests.DrainProvider, which appends those
	// page params, so a `limit` here 400s every single rect query.
	mapMonstersRectResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters/in-rect?x1=%d&y1=%d&x2=%d&y2=%d&max=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

// inMapUrl returns the list URL for the monsters currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(field field.Model) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance())
}

// inMapRectUrl returns the list URL for the atlas-monsters rectangle query.
// Bounds are inclusive; limit == 0 means "no cap". Bare URL (not a
// requests.Request) because the list is paginated server-side and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size] params.
func inMapRectUrl(f field.Model, x1, y1, x2, y2 int16, limit uint32) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersRectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String(), x1, y1, x2, y2, limit)
}

func requestCreate(field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()), m)
}
