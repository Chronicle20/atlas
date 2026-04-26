package paginate_test

import (
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/stretchr/testify/assert"
)

func TestLastPage_NormalCases(t *testing.T) {
	assert.Equal(t, 1, paginate.Envelope{Total: 0, PageSize: 50}.LastPage())
	assert.Equal(t, 1, paginate.Envelope{Total: 1, PageSize: 50}.LastPage())
	assert.Equal(t, 1, paginate.Envelope{Total: 50, PageSize: 50}.LastPage())
	assert.Equal(t, 2, paginate.Envelope{Total: 51, PageSize: 50}.LastPage())
	assert.Equal(t, 25, paginate.Envelope{Total: 1234, PageSize: 50}.LastPage())
}

func TestMeta_Shape(t *testing.T) {
	env := paginate.Envelope{Total: 1234, PageNumber: 3, PageSize: 50}
	m := env.Meta()
	assert.Equal(t, 1234, m["total"])
	page := m["page"].(map[string]interface{})
	assert.Equal(t, 3, page["number"])
	assert.Equal(t, 50, page["size"])
	assert.Equal(t, 25, page["last"])
}

func TestBuildLinks_MiddlePage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?filter[compartment]=equipment&page[number]=3&page[size]=50", nil)
	env := paginate.Envelope{Total: 1234, PageNumber: 3, PageSize: 50}
	links := env.BuildLinks(req)

	assert.Contains(t, links, "self")
	assert.Contains(t, links, "first")
	assert.Contains(t, links, "prev")
	assert.Contains(t, links, "next")
	assert.Contains(t, links, "last")

	assert.Contains(t, links["self"].Href, "page%5Bnumber%5D=3")
	assert.Contains(t, links["first"].Href, "page%5Bnumber%5D=1")
	assert.Contains(t, links["prev"].Href, "page%5Bnumber%5D=2")
	assert.Contains(t, links["next"].Href, "page%5Bnumber%5D=4")
	assert.Contains(t, links["last"].Href, "page%5Bnumber%5D=25")
	assert.Contains(t, links["self"].Href, "filter%5Bcompartment%5D=equipment")
	assert.Contains(t, links["self"].Href, "page%5Bsize%5D=50")
}

func TestBuildLinks_FirstPage_PrevOmitted(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?page[number]=1&page[size]=50", nil)
	env := paginate.Envelope{Total: 1234, PageNumber: 1, PageSize: 50}
	links := env.BuildLinks(req)
	_, hasPrev := links["prev"]
	assert.False(t, hasPrev)
	_, hasNext := links["next"]
	assert.True(t, hasNext)
}

func TestBuildLinks_LastPage_NextOmitted(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?page[number]=25&page[size]=50", nil)
	env := paginate.Envelope{Total: 1234, PageNumber: 25, PageSize: 50}
	links := env.BuildLinks(req)
	_, hasNext := links["next"]
	assert.False(t, hasNext)
	_, hasPrev := links["prev"]
	assert.True(t, hasPrev)
}

func TestBuildLinks_SinglePage_NoPrevNoNext(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?page[number]=1&page[size]=50", nil)
	env := paginate.Envelope{Total: 5, PageNumber: 1, PageSize: 50}
	links := env.BuildLinks(req)
	_, hasPrev := links["prev"]
	_, hasNext := links["next"]
	assert.False(t, hasPrev)
	assert.False(t, hasNext)
	assert.Contains(t, links["first"].Href, "page%5Bnumber%5D=1")
	assert.Contains(t, links["last"].Href, "page%5Bnumber%5D=1")
}

func TestBuildLinks_PastEnd_PrevPointsToLast(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?page[number]=999&page[size]=50", nil)
	env := paginate.Envelope{Total: 1234, PageNumber: 999, PageSize: 50}
	links := env.BuildLinks(req)
	assert.Contains(t, links["self"].Href, "page%5Bnumber%5D=999")
	assert.Contains(t, links["prev"].Href, "page%5Bnumber%5D=25")
	_, hasNext := links["next"]
	assert.False(t, hasNext)
	assert.Contains(t, links["last"].Href, "page%5Bnumber%5D=25")
}

func TestBuildLinks_DefaultsFillIn(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data/item-strings?search=bow", nil)
	env := paginate.Envelope{Total: 100, PageNumber: 1, PageSize: 50}
	links := env.BuildLinks(req)
	assert.Contains(t, links["self"].Href, "page%5Bnumber%5D=1")
	assert.Contains(t, links["self"].Href, "page%5Bsize%5D=50")
	assert.Contains(t, links["self"].Href, "search=bow")
}
