package wish

import (
	"fmt"
	"net/url"

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

// requestByCharacterAndType fetches only the character's cart or wanted entries
// (atlas-mts handleGetCharacterWishlist honors the `type` query param).
func requestByCharacterAndType(characterId uint32, wishType string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId) + "?type=" + url.QueryEscape(wishType))
}
