package location

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/location"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestByCharacterId(characterId character.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
