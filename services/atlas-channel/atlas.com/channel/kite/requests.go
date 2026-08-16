package kite

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource carries the /instances/{instanceId} segment from day one — the
// chalkboard sibling shipped without it and 404'd against the real route for
// its entire life (chalkboard/requests.go:9-17).
const Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/kites"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "KITES")
}

// inMapUrl returns the list URL for the kites currently displayed in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// paginated server-side and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func inMapUrl(ctx context.Context, f field.Model) string {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}
