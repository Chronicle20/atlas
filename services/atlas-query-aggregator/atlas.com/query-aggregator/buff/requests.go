package buff

import (
	"atlas-query-aggregator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CharactersResource = "characters"
	BuffsResource      = "buffs"
	ByCharacter        = CharactersResource + "/%d/" + BuffsResource
)

func getBaseRequest() string {
	return requests.RootUrl("BUFFS")
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, characterId))
}
