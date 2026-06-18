package listing

import (
	"strings"
	"testing"
)

// TestBrowseFilterQuery asserts the BrowseFilter renders only its set fields into
// the query string (empty/zero fields are omitted so atlas-mts treats them as
// unfiltered).
func TestBrowseFilterQuery(t *testing.T) {
	if got := (BrowseFilter{}).query(); got != "" {
		t.Errorf("empty filter: want no query, got %q", got)
	}

	f := BrowseFilter{SellerName: "Bob", Page: 2, PageSize: 16, ItemId: 1302000}
	got := f.query()
	for _, want := range []string{"sellerName=Bob", "page=2", "pageSize=16", "itemId=1302000"} {
		if !strings.Contains(got, want) {
			t.Errorf("query %q missing %q", got, want)
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
