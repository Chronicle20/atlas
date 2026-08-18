package merchant

import (
	"context"
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

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MERCHANT")
}

func requestShop(ctx context.Context, shopId string) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ShopResource+"?include=listings", shopId))
}

// inFieldUrl returns the list URL for shops on a field. It is a bare URL
// (not a requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func inFieldUrl(ctx context.Context, f field.Model) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}

// byCharacterIdUrl is the bare-URL sibling of inFieldUrl for the
// per-character shop list (task-117).
func byCharacterIdUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+CharacterResource, characterId), nil
}

func requestVisiting(ctx context.Context, characterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+VisitingResource, characterId))
}

func requestSearchListings(ctx context.Context, itemId uint32, worldId world.Id, descending bool) requests.Request[[]ListingSearchRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]ListingSearchRestModel](err)
	}
	order := "asc"
	if descending {
		order = "desc"
	}
	return requests.GetRequest[[]ListingSearchRestModel](fmt.Sprintf(root+SearchListingsResource, itemId, worldId, order))
}

func requestTopSearches(ctx context.Context, worldId world.Id) requests.Request[[]TopSearchRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]TopSearchRestModel](err)
	}
	return requests.GetRequest[[]TopSearchRestModel](fmt.Sprintf(root+TopSearchesResource, worldId))
}

func requestFrederickStatus(ctx context.Context, characterId uint32) requests.Request[FrederickStatusRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[FrederickStatusRestModel](err)
	}
	return requests.GetRequest[FrederickStatusRestModel](fmt.Sprintf(root+FrederickResource, characterId))
}

// blacklistUrl is the bare-URL sibling of inFieldUrl for the per-shop
// blacklist list, now paginated server-side (task-117) and consumed via
// requests.DrainProvider — the mini-room dialog shows the whole blacklist.
func blacklistUrl(ctx context.Context, shopId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+BlacklistResource, shopId), nil
}

// visitsUrl is the bare-URL sibling of blacklistUrl for the per-shop visit
// log (task-117): the log grows with unique visitor names, so the dialog
// consumer drains every page.
func visitsUrl(ctx context.Context, shopId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+VisitsResource, shopId), nil
}
