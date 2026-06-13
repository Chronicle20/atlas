package mount

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ByCharacterResource = "characters/%d/mount"
)

func getBaseRequest() string {
	return requests.RootUrl("MOUNTS")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterResource, characterId))
}
