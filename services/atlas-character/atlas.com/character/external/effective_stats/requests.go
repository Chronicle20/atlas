package effective_stats

import (
	"atlas-character/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/characters/%d/stats"
)

func getBaseRequest() string {
	return requests.RootUrl("EFFECTIVE_STATS")
}

// RequestByCharacter returns a request to fetch effective stats for a character
func RequestByCharacter(worldId byte, channelId byte, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId, channelId, characterId))
}
