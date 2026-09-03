package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the path template for listing the scripted NPCs currently
// placed on one map instance, per atlas-maps' map/npc.InitResource route
// (services/atlas-maps/atlas.com/maps/map/npc/resource.go).
const Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/npcs"

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

// inMapUrl returns the list URL for the scripted NPCs currently placed on
// one map instance. It is a bare URL (not a requests.Request) because the
// list is consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request -- atlas-maps' own
// endpoint sends no pagination envelope, so DrainProvider treats the
// single response as the complete collection.
func inMapUrl(ctx context.Context, f field.Model) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default. The
// injected closure ignores ctx -- tests always exercise the fixed httptest
// URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
