package playernpc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the path template for listing the Player NPCs currently
// deployed in one map, per atlas-player-npcs' PRD §5 list filters
// (services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource.go's
// parseListFilters).
const Resource = "player-npcs?filter[mapId]=%d&filter[worldId]=%d"

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PLAYER_NPCS")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

// inMapUrl returns the list URL for the Player NPCs currently deployed in
// one map. It is a bare URL (not a requests.Request) because the list is
// paginated server-side and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func inMapUrl(ctx context.Context, f field.Model) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, f.MapId(), f.WorldId()), nil
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
