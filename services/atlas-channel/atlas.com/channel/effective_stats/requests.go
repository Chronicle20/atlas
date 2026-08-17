package effective_stats

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const Resource = "worlds/%d/channels/%d/characters/%d/stats"

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "EFFECTIVE_STATS")
}

// requestByCharacter fetches effective stats for a character from the
// atlas-effective-stats service. World+channel are needed because
// effective stats depend on session-side context (channel-scoped buffs).
func requestByCharacter(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel] {
	root, err := baseURLProvider(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, worldId, channelId, characterId))
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
