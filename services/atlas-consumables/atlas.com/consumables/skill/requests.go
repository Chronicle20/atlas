package skill

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/skills"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

// characterSkillsUrl returns the list URL for a character's skills. The
// atlas-skills list endpoint is paginated (task-117); it is consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func characterSkillsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}
