package rate

import (
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestByCharacter(worldId byte, channelId byte, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId, channelId, characterId))
}
