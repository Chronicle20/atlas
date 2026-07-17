package listing

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestBrowseFilterQuery asserts the BrowseFilter renders only its set fields into
// the query string (empty/zero fields are omitted so atlas-mts treats them as
// unfiltered).
func TestBrowseFilterQuery(t *testing.T) {
	if got := (BrowseFilter{}).query(); got != "" {
		t.Errorf("empty filter: want no query, got %q", got)
	}

	// Page is 0-based (mirrors the game client); query() must render it onto
	// the wire's 1-based page[number] as Page+1 (task-117's off-by-one).
	f := BrowseFilter{SellerName: "Bob", Page: 2, PageSize: 16, ItemId: 1302000}
	got := f.query()
	for _, want := range []string{"sellerName=Bob", "page%5Bnumber%5D=3", "page%5Bsize%5D=16", "itemId=1302000"} {
		if !strings.Contains(got, want) {
			t.Errorf("query %q missing %q", got, want)
		}
	}
	// The old bare 0-based params must never appear.
	for _, absent := range []string{"page=2", "pageSize=16"} {
		if strings.Contains(got, absent) {
			t.Errorf("query %q should not contain legacy param %q", got, absent)
		}
	}
	if !strings.HasPrefix(got, "?") {
		t.Errorf("query must start with '?': %q", got)
	}
	// category/subCategory/saleType were unset -> must be absent
	for _, absent := range []string{"category=", "subCategory=", "saleType="} {
		if strings.Contains(got, absent) {
			t.Errorf("query %q should not contain %q", got, absent)
		}
	}
}

// TestBrowseFilterQuerySerial asserts the serial filter renders when set. The
// zzim/wish remove arms resolve a listing by its ITC serial through this filter
// (GetBySerial), so the param must reach atlas-mts.
func TestBrowseFilterQuerySerial(t *testing.T) {
	got := (BrowseFilter{Serial: 4242}).query()
	if !strings.Contains(got, "serial=4242") {
		t.Errorf("query %q missing serial=4242", got)
	}
	// Serial 0 means "unset" -> must be omitted.
	if strings.Contains((BrowseFilter{}).query(), "serial=") {
		t.Errorf("empty filter must not contain serial=")
	}
}

// TestBrowseFilterQuerySellerId asserts the sellerId filter renders when set. The
// ENTER_MTS "my sales" announce browses with this filter so only the entering
// character's own active listings are returned (GET_USER_SALE_ITEM_DONE).
func TestBrowseFilterQuerySellerId(t *testing.T) {
	got := (BrowseFilter{SellerId: 100100}).query()
	if !strings.Contains(got, "sellerId=100100") {
		t.Errorf("query %q missing sellerId=100100", got)
	}
	// SellerId 0 means "unset" -> must be omitted.
	if strings.Contains((BrowseFilter{}).query(), "sellerId=") {
		t.Errorf("empty filter must not contain sellerId=")
	}
}

// TestBrowseFilterQueryCategory asserts the category/subCategory/saleType filters
// render when set (the browse arm passes them through to atlas-mts equality
// filters).
func TestBrowseFilterQueryCategory(t *testing.T) {
	f := BrowseFilter{Category: "equip", SubCategory: "onehand", SaleType: "auction"}
	got := f.query()
	for _, want := range []string{"category=equip", "subCategory=onehand", "saleType=auction"} {
		if !strings.Contains(got, want) {
			t.Errorf("query %q missing %q", got, want)
		}
	}
}

// TestBrowseFilterQueryTemplateIds asserts the TemplateIds filter renders as a
// comma-joined `itemIds` param (the marketplace name-search result set). An empty
// slice must omit the param so atlas-mts treats it as unfiltered.
func TestBrowseFilterQueryTemplateIds(t *testing.T) {
	got := (BrowseFilter{TemplateIds: []uint32{1302000, 1302001, 1402000}}).query()
	// url.Values.Encode escapes the comma to %2C; assert on the encoded form.
	if !strings.Contains(got, "itemIds=1302000%2C1302001%2C1402000") {
		t.Errorf("query %q missing comma-joined itemIds", got)
	}
	if strings.Contains((BrowseFilter{}).query(), "itemIds=") {
		t.Errorf("empty filter must not contain itemIds=")
	}
	if strings.Contains((BrowseFilter{TemplateIds: []uint32{}}).query(), "itemIds=") {
		t.Errorf("empty TemplateIds slice must not contain itemIds=")
	}
}

// TestBrowseUrlOmitsPageParams asserts browseUrl (the BrowseAll/DrainProvider
// URL builder) never bakes in page params: requests.DrainProvider appends its
// own page[number]/page[size] each iteration, so a baked-in value here would
// pin every drained page to the same window instead of advancing.
func TestBrowseUrlOmitsPageParams(t *testing.T) {
	got := browseUrl(world.Id(1), BrowseFilter{SellerId: 100100})
	if strings.Contains(got, "page") {
		t.Errorf("browseUrl %q must not contain any page param", got)
	}
	if !strings.Contains(got, "sellerId=100100") {
		t.Errorf("browseUrl %q missing sellerId=100100", got)
	}
}
