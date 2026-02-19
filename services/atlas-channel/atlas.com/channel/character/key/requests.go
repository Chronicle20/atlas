package key

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/keys"
	ByKey    = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("KEYS")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

func updateKey(characterId uint32, key int32, theType int8, action int32) requests.Request[RestModel] {
	i := RestModel{
		Key:    key,
		Type:   theType,
		Action: action,
	}

	return requests.PatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByKey, characterId, key), i)
}
