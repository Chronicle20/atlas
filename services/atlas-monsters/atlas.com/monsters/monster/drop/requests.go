package drop

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monsterDropsResource = "monsters/%d/drops"
)

func getBaseRequest() string {
	return requests.RootUrl("DROPS_INFORMATION")
}

// monsterDropsUrl is a bare URL (not a requests.Request) because the list is
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func monsterDropsUrl(monsterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+monsterDropsResource, monsterId)
}
