package holding

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the atlas-mts take-home holding read endpoint:
// GET /characters/{characterId}/mts/holding. It mirrors atlas-mts's
// holding.handleGetCharacterHoldings.
const Resource = "characters/%d/mts/holding"

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

// byCharacterUrl returns the list URL for a character's take-home holdings. It
// is a bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func byCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}
