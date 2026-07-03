package skill

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/skills"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

// characterSkillsUrl returns the list URL for a character's skills. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterSkillsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

func requestById(characterId uint32, id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, characterId, id))
}
