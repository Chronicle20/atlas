package effectivestats

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ByCharacter = "worlds/%d/channels/%d/characters/%d/stats"
)

func getBaseRequest() string {
	return requests.RootUrl("EFFECTIVE_STATS")
}

func RequestByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, worldId, channelId, characterId))
}
