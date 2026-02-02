package rates

import (
	"atlas-saga-orchestrator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	ratesPath = "rates/%d/%d/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestRates(worldId byte, channelId byte, characterId uint32) requests.Request[DataContainer] {
	return rest.MakeGetRequest[DataContainer](fmt.Sprintf(getBaseRequest()+ratesPath, worldId, channelId, characterId))
}
