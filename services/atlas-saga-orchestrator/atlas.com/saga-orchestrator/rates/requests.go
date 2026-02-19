package rates

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	ratesPath = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestRates(ch channel.Model, characterId uint32) requests.Request[DataContainer] {
	return requests.GetRequest[DataContainer](fmt.Sprintf(getBaseRequest()+ratesPath, ch.WorldId(), ch.Id(), characterId))
}
