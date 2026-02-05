package rates

import (
	"atlas-saga-orchestrator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	ratesPath = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestRates(worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[DataContainer] {
	return rest.MakeGetRequest[DataContainer](fmt.Sprintf(getBaseRequest()+ratesPath, worldId, channelId, characterId))
}
