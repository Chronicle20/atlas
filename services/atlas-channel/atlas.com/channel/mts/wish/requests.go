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

// WorldResource is the atlas-mts cross-character want-ad endpoint:
// GET /worlds/{worldId}/mts/wishlist. It mirrors atlas-mts's
// wish.handleGetWorldWishlist (every want-ad in a world, all characters).
const WorldResource = "worlds/%d/mts/wishlist"

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// requestWantedByWorld fetches every want-ad in a world, across all characters
// (atlas-mts handleGetWorldWishlist returns the type=wanted entries world-wide).
func requestWantedByWorld(worldId byte) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+WorldResource, worldId))
}

// requestByCharacterAndType fetches only the character's cart or wanted entries
// (atlas-mts handleGetCharacterWishlist honors the `type` query param).
func requestByCharacterAndType(characterId uint32, wishType string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId) + "?type=" + url.QueryEscape(wishType))
}
