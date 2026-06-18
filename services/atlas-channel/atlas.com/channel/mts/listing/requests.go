package listing

import (
	"fmt"
	"net/url"
	"strconv"

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
	SellerName  string
	Page        int
	PageSize    int
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
