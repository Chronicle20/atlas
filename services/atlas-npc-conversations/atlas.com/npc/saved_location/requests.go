package saved_location

import (
	"atlas-npc-conversations/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource           = "locations"
	ByCharacterAndType = "/characters/%d/" + Resource + "/%s"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestByCharacterAndType(characterId uint32, locationType string) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterAndType, characterId, locationType))
}
