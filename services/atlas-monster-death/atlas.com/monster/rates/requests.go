package rates

import (
	"atlas-monster-death/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	RatesResource = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestForCharacter(worldId byte, channelId byte, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+RatesResource, worldId, channelId, characterId))
}
