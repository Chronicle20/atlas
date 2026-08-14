package pendingchange

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/pending-changes"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestCreateUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

// requestCancelUrl points at the self-scoped cancel route (task-227
// client-cancel addendum) -- a fixed sub-path, not an {id}, since the wire
// packet that drives it carries no pending-change id.
func requestCancelUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource+"/cancel", characterId)
}
