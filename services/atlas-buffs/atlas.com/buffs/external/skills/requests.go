package skills

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource         = "characters/%d/skills"
	ByCharacterSkill = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

func RequestByCharacterAndSkill(characterId uint32, skillId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterSkill, characterId, skillId))
}
