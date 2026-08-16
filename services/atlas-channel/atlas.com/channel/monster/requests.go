package monster

import (
	"context"
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
	monstersResource        = "monsters/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTERS")
}

// inMapUrl returns the list URL for the monsters currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(ctx context.Context, f field.Model) string {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}

// inMapRectUrl returns the list URL for the atlas-monsters rectangle query
// used for AoE skill targeting (e.g., Priest Doom). Bounds are inclusive;
// limit == 0 means "no cap". Bare URL for the same reason as inMapUrl --
// atlas-monsters preserves its ascending-distance-from-center order across
// pages, so draining is still safe (page order is meaningful, not
// re-sorted).
func inMapRectUrl(ctx context.Context, f field.Model, x1, y1, x2, y2 int16, limit uint32) string  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+mapMonstersRectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String(), x1, y1, x2, y2, limit), nil
}

func requestById(ctx context.Context, uniqueId uint32) requests.Request[RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+monstersResource, uniqueId))
}
