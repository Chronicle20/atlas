package effective_stats

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const Resource = "worlds/%d/channels/%d/characters/%d/stats"

var baseURLProvider = func() string {
	return requests.RootUrl("EFFECTIVE_STATS")
}

// requestByCharacter fetches effective stats for a character from the
// atlas-effective-stats service. World+channel are needed because
// effective stats depend on session-side context (channel-scoped buffs).
func requestByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(baseURLProvider()+Resource, worldId, channelId, characterId))
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
