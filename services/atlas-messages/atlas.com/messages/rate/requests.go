package rate

import (
	"atlas-messages/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest() string {
	return requests.RootUrl("RATES")
}

func requestByCharacter(ch channel.Model, characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, ch.WorldId(), ch.Id(), characterId))
}
