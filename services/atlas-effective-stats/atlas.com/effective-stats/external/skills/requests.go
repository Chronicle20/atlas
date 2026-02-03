package skills

import (
	"atlas-effective-stats/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource           = "characters/%d/skills"
	CharacterSkillsAll = Resource
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

// RequestCharacterSkills returns a request to fetch all skills for a character
func RequestCharacterSkills(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+CharacterSkillsAll, characterId))
}
