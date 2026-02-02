package rate

import (
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "rates/%d/%d/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestByCharacter(worldId byte, channelId byte, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId, channelId, characterId))
}
