package rates

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	RatesResource = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestForCharacter(ch channel.Model, characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+RatesResource, ch.WorldId(), ch.Id(), characterId))
}
