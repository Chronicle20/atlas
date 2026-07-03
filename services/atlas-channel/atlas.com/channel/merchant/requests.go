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
)

func getBaseRequest() string {
	return requests.RootUrl("MERCHANT")
}

func requestShop(shopId string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ShopResource, shopId))
}

func requestInField(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+CharacterResource, characterId))
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
