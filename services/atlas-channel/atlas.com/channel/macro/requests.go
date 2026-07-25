package macro

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/macros"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

// characterMacrosUrl returns the list URL for a character's skill macros. It
// is a bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterMacrosUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}
