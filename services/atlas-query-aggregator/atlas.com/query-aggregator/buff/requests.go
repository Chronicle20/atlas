package buff

import (
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
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, characterId))
}
