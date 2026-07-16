package skill

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterAndId, characterId, skillId))
}

// byCharacterUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func byCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+ByCharacter, characterId)
}
