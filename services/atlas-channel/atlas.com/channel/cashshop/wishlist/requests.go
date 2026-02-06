package wishlist

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource = "characters/%d/cash-shop/wishlist"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

func addForCharacterId(characterId uint32, serialNumber uint32) requests.Request[RestModel] {
	i := RestModel{
		Id:           uuid.Nil,
		CharacterId:  characterId,
		SerialNumber: serialNumber,
	}
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId), i)
}

func clearForCharacterId(characterId uint32) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
