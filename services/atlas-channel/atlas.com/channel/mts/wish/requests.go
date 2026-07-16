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

// byCharacterUrl returns the list URL for a character's wishlist. It is a bare
// URL (not a requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func byCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

// wantedByWorldUrl returns the list URL for every want-ad in a world, across all
// characters (atlas-mts handleGetWorldWishlist returns the type=wanted entries
// world-wide). Paginated server-side (task-117); consumed via DrainProvider.
func wantedByWorldUrl(worldId byte) string {
	return fmt.Sprintf(getBaseRequest()+WorldResource, worldId)
}

// byCharacterAndTypeUrl returns the list URL for only the character's cart or
// wanted entries (atlas-mts handleGetCharacterWishlist honors the `type` query
// param). Paginated server-side (task-117); consumed via DrainProvider.
func byCharacterAndTypeUrl(characterId uint32, wishType string) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId) + "?type=" + url.QueryEscape(wishType)
}
