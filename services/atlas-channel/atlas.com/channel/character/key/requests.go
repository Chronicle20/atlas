package key

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/keys"
	ByKey    = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("KEYS")
}

// characterKeysUrl returns the list URL for a character's key map. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterKeysUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

func updateKey(characterId uint32, key int32, theType int8, action int32) requests.Request[RestModel] {
	i := RestModel{
		Key:    key,
		Type:   theType,
		Action: action,
	}

	return requests.PatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByKey, characterId, key), i)
}
