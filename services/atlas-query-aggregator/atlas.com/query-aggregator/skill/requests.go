package skill

import (
	"atlas-query-aggregator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CharactersResource = "characters"
	SkillsResource     = "skills"
	ByCharacterAndId   = CharactersResource + "/%d/" + SkillsResource + "/%d"
	ByCharacter        = CharactersResource + "/%d/" + SkillsResource
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

func requestById(characterId uint32, skillId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterAndId, characterId, skillId))
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, characterId))
}
