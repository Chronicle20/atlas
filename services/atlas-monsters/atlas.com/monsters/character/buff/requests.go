package buff

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const characterBuffsResource = "characters/%d/buffs"

func getBaseRequest() string {
	return requests.RootUrl("BUFFS")
}

// characterBuffsUrl is a bare URL because atlas-buffs' list is paginated
// (task-117) and consumed via requests.DrainProvider.
func characterBuffsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+characterBuffsResource, characterId)
}
