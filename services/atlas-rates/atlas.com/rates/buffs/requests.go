package buffs

import (
	"atlas-rates/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/buffs"
)

func getBaseRequest() string {
	return requests.RootUrl("BUFFS")
}

func requestBuffs(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
