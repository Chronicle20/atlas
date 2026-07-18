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
// sellerName/page[number]/page[size]). It mirrors atlas-mts's
// listing.handleBrowseListings, which pages via the repo-wide
// page[number](1-based)/page[size] convention (task-117).
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
	// Page is the 0-based page index (mirrors the game client's own 0-based
	// paging field). query() renders it onto the wire's 1-based page[number]
	// (Page+1); the caller never sees the +1 — it stays an implementation
	// detail of the single-page Browse/BrowseProvider path. Unused by the
	// semantic-all BrowseAll path, which pages via requests.DrainProvider
	// against the filter-only URL (browseUrl) instead.
	Page     int
	PageSize int
}

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

// filterQuery renders only the non-paging filter fields. Shared by query()
// (the single-page Browse path, which layers page[number]/page[size] on
// top) and browseUrl() (the BrowseAll/DrainProvider path, which appends its
// own page params per iteration and must not have Page/PageSize baked in).
func (f BrowseFilter) filterQuery() url.Values {
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
	return q
}

// query renders the filter as a single-page browse URL query string (leading
// "?") or "" when empty. Page (0-based) is rendered onto the wire's 1-based
// page[number] as Page+1 — atlas-mts's page[number] defaults to 1 (its own
// 0-based equivalent, page 0), so a zero-valued Page is omitted exactly as
// the pre-task-117 bare "page" param was, preserving the same default
// window. PageSize renders directly onto page[size] (same units, no
// conversion) only when the caller set a non-default value.
func (f BrowseFilter) query() string {
	q := f.filterQuery()
	if f.Page != 0 {
		q.Set("page[number]", strconv.Itoa(f.Page+1))
	}
	if f.PageSize != 0 {
		q.Set("page[size]", strconv.Itoa(f.PageSize))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

func requestBrowse(worldId world.Id, f BrowseFilter) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, byte(worldId)) + f.query())
}

// browseUrl returns the bare browse URL for the given world/filters, WITHOUT
// any page params, for requests.DrainProvider (BrowseAll): DrainProvider
// appends its own page[number]/page[size] per iteration, so this must not
// bake in Page/PageSize (BrowseAll callers never set them anyway).
func browseUrl(worldId world.Id, f BrowseFilter) string {
	base := fmt.Sprintf(getBaseRequest()+Resource, byte(worldId))
	q := f.filterQuery()
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}
