package listing

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the atlas-mts browse endpoint: GET /worlds/{worldId}/listings with
// the browse filter as query params (category/subCategory/saleType/itemId/
// sellerName/page/pageSize). It mirrors atlas-mts's listing.handleBrowseListings.
const Resource = "worlds/%d/listings"

// BrowseFilter carries the optional browse/search query parameters decoded from
// the GET_ITC_LIST / SEARCH_ITC_LIST ITC_OPERATION arms. Empty/zero fields are
// omitted from the query string (atlas-mts treats absent params as unfiltered).
type BrowseFilter struct {
	Category    string
	SubCategory string
	SaleType    string
	ItemId      uint32
	// TemplateIds restricts the browse to this set of item template ids (the
	// resolved hits of a SEARCH_ITC_LIST name search). When non-empty it renders
	// the comma-joined `itemIds` param, which atlas-mts maps to `template_id IN (?)`.
	// An empty (but non-nil) slice means "name search matched nothing" — the search
	// arm short-circuits before browsing in that case, so it is never rendered here.
	TemplateIds []uint32
	Serial      uint32
	// Serials restricts the browse to this set of ITC serials (rendered as the
	// comma-joined `serials` param -> atlas-mts `serial IN (?)`). The Cart uses it to
	// resolve all its favorited listings in ONE browse instead of a per-entry
	// GetBySerial (avoids an N+1 on every cart re-push).
	Serials         []uint32
	SellerId        uint32
	ExcludeSellerId uint32 // public-browse filter: omit this seller's own listings
	SellerName      string
	// OfferWishSerial restricts the browse to `offer` listings made on a specific
	// want-ad (VIEW_WISH shows a poster the offers on their ad).
	OfferWishSerial uint32
	// ExcludeOffers omits sale_type=offer rows from a public browse so escrowed
	// offers never appear in the For-Sale / Auction tabs.
	ExcludeOffers bool
	Page          int
	PageSize      int
}

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

// query renders the filter as a URL query string (leading "?") or "" when empty.
func (f BrowseFilter) query() string {
	q := url.Values{}
	if f.Category != "" {
		q.Set("category", f.Category)
	}
	if f.SubCategory != "" {
		q.Set("subCategory", f.SubCategory)
	}
	if f.SaleType != "" {
		q.Set("saleType", f.SaleType)
	}
	if f.ItemId != 0 {
		q.Set("itemId", strconv.FormatUint(uint64(f.ItemId), 10))
	}
	if len(f.TemplateIds) > 0 {
		parts := make([]string, 0, len(f.TemplateIds))
		for _, id := range f.TemplateIds {
			parts = append(parts, strconv.FormatUint(uint64(id), 10))
		}
		q.Set("itemIds", strings.Join(parts, ","))
	}
	if f.Serial != 0 {
		q.Set("serial", strconv.FormatUint(uint64(f.Serial), 10))
	}
	if len(f.Serials) > 0 {
		parts := make([]string, 0, len(f.Serials))
		for _, sn := range f.Serials {
			parts = append(parts, strconv.FormatUint(uint64(sn), 10))
		}
		q.Set("serials", strings.Join(parts, ","))
	}
	if f.SellerId != 0 {
		q.Set("sellerId", strconv.FormatUint(uint64(f.SellerId), 10))
	}
	if f.ExcludeSellerId != 0 {
		q.Set("excludeSellerId", strconv.FormatUint(uint64(f.ExcludeSellerId), 10))
	}
	if f.OfferWishSerial != 0 {
		q.Set("offerWishSerial", strconv.FormatUint(uint64(f.OfferWishSerial), 10))
	}
	if f.ExcludeOffers {
		q.Set("excludeOffers", "true")
	}
	if f.SellerName != "" {
		q.Set("sellerName", f.SellerName)
	}
	if f.Page != 0 {
		q.Set("page", strconv.Itoa(f.Page))
	}
	if f.PageSize != 0 {
		q.Set("pageSize", strconv.Itoa(f.PageSize))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

func requestBrowse(worldId world.Id, f BrowseFilter) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, byte(worldId)) + f.query())
}
