package wishlist

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource = "characters/%d/cash-shop/wishlist"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

// byCharacterIdUrl returns the list URL for a character's cash-shop
// wishlist. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func byCharacterIdUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

func addForCharacterId(characterId uint32, serialNumber uint32) requests.Request[RestModel] {
	i := RestModel{
		Id:           uuid.Nil,
		CharacterId:  characterId,
		SerialNumber: serialNumber,
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId), i)
}

func clearForCharacterId(characterId uint32) requests.EmptyBodyRequest {
	return requests.DeleteRequest(fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
