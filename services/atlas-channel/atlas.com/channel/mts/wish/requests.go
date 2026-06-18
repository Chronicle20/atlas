package wish

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the atlas-mts wish-list read endpoint:
// GET /characters/{characterId}/mts/wishlist. It mirrors atlas-mts's
// wish.handleGetCharacterWishlist.
const Resource = "characters/%d/mts/wishlist"

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
