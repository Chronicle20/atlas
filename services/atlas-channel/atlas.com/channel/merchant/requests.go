package merchant

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource               = "worlds/%d/channels/%d/maps/%d/instances/%s/merchants"
	ShopResource           = "merchants/%s"
	CharacterResource      = "characters/%d/merchants"
	VisitingResource       = "characters/%d/visiting"
	SearchListingsResource = "merchants/search/listings?itemId=%d&worldId=%d&order=%s"
	TopSearchesResource    = "worlds/%d/shop-searches/top"
	FrederickResource      = "characters/%d/frederick"
	BlacklistResource      = "merchants/%s/blacklist"
	VisitsResource         = "merchants/%s/visits"
)

func getBaseRequest() string {
	return requests.RootUrl("MERCHANT")
}

func requestShop(shopId string) requests.Request[RestModel] {
	// include=listings is load-bearing: atlas-merchant gates the shop's listing
	// data behind the JSON:API include, and GetShop feeds every shop-view
	// refresh (buildShopItems(shop.Listings()) -> UPDATE_MERCHANT) plus the
	// enter/maintenance views. Without it the store renders empty even when
	// listings exist (task-127).
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ShopResource+"?include=listings", shopId))
}

// inFieldUrl returns the list URL for shops on a field. It is a bare URL
// (not a requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func inFieldUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// byCharacterIdUrl is the bare-URL sibling of inFieldUrl for the
// per-character shop list (task-117).
func byCharacterIdUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+CharacterResource, characterId)
}

func requestVisiting(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+VisitingResource, characterId))
}

func requestSearchListings(itemId uint32, worldId world.Id, descending bool) requests.Request[[]ListingSearchRestModel] {
	order := "asc"
	if descending {
		order = "desc"
	}
	return requests.GetRequest[[]ListingSearchRestModel](fmt.Sprintf(getBaseRequest()+SearchListingsResource, itemId, worldId, order))
}

func requestTopSearches(worldId world.Id) requests.Request[[]TopSearchRestModel] {
	return requests.GetRequest[[]TopSearchRestModel](fmt.Sprintf(getBaseRequest()+TopSearchesResource, worldId))
}

func requestFrederickStatus(characterId uint32) requests.Request[FrederickStatusRestModel] {
	return requests.GetRequest[FrederickStatusRestModel](fmt.Sprintf(getBaseRequest()+FrederickResource, characterId))
}

func requestBlacklist(shopId string) requests.Request[[]BlacklistRestModel] {
	return requests.GetRequest[[]BlacklistRestModel](fmt.Sprintf(getBaseRequest()+BlacklistResource, shopId))
}

func requestVisits(shopId string) requests.Request[[]VisitRestModel] {
	return requests.GetRequest[[]VisitRestModel](fmt.Sprintf(getBaseRequest()+VisitsResource, shopId))
}
