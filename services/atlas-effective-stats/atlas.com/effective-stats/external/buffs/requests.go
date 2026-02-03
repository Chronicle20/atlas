package buffs

import (
	"atlas-effective-stats/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CharacterBuffs = "characters/%d/buffs"
)

func getBaseRequest() string {
	return requests.RootUrl("BUFFS")
}

// RequestCharacterBuffs returns a request to fetch active buffs for a character
func RequestCharacterBuffs(characterId uint32) requests.Request[[]BuffRestModel] {
	return rest.MakeGetRequest[[]BuffRestModel](fmt.Sprintf(getBaseRequest()+CharacterBuffs, characterId))
}
