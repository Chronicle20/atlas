package session

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	SessionsResource = "characters/%d/sessions"
	PlaytimeResource = "characters/%d/sessions/playtime"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTER")
}

// RequestSessionsSince fetches sessions since the given Unix timestamp
func RequestSessionsSince(characterId uint32, sinceUnix int64) requests.Request[[]SessionRestModel] {
	url := fmt.Sprintf(getBaseRequest()+SessionsResource+"?since=%d", characterId, sinceUnix)
	return requests.GetRequest[[]SessionRestModel](url)
}

// RequestPlaytimeSince fetches computed playtime since the given Unix timestamp
