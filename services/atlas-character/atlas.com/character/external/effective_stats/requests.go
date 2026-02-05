package effective_stats

import (
	"atlas-character/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/characters/%d/stats"
)

func getBaseRequest() string {
	return requests.RootUrl("EFFECTIVE_STATS")
}

// RequestByCharacter returns a request to fetch effective stats for a character
func RequestByCharacter(ch channel.Model, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, ch.WorldId(), ch.Id(), characterId))
}
