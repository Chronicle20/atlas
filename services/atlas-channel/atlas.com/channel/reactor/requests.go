package reactor

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/reactors"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "REACTORS")
}

// inMapUrl returns the list URL for the reactors currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(ctx context.Context, f field.Model) string {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}
