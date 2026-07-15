package session

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	SessionsResource = "characters/%d/sessions"
	PlaytimeResource = "characters/%d/sessions/playtime"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTER")
}

// SessionsSinceUrl builds the sessions endpoint URL for a character, filtered
// to sessions since the given Unix timestamp. The endpoint paginates, so
// callers that need the whole since-filtered collection must drain every
// page (see requests.DrainProvider) rather than issuing a single GET.
func SessionsSinceUrl(characterId uint32, sinceUnix int64) string {
	return fmt.Sprintf(getBaseRequest()+SessionsResource+"?since=%d", characterId, sinceUnix)
}

// RequestPlaytimeSince fetches computed playtime since the given Unix timestamp
