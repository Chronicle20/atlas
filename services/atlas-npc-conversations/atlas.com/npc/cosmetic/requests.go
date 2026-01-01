package cosmetic

import (
	"atlas-npc-conversations/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CharacterResource = "/characters/%d"
)

func getCharacterServiceUrl() string {
	return requests.RootUrl("CHARACTERS")
}

func requestCharacterById(characterId uint32) requests.Request[RestCharacterModel] {
	return rest.MakeGetRequest[RestCharacterModel](fmt.Sprintf(getCharacterServiceUrl()+CharacterResource, characterId))
}

func requestUpdateCharacter(characterId uint32, updateRequest CharacterUpdateRequest) requests.Request[RestCharacterModel] {
	return rest.MakePatchRequest[RestCharacterModel](
		fmt.Sprintf(getCharacterServiceUrl()+CharacterResource, characterId),
		updateRequest,
	)
}
