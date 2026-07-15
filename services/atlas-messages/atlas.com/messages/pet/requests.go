package pet

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource      = "pets"
	ByCharacterId = "/characters/%d/" + Resource
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

// byCharacterIdUrl returns the list URL for a character's pets. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func byCharacterIdUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+ByCharacterId, characterId)
}
