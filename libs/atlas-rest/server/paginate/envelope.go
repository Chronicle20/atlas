package paginate

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// Envelope captures everything needed to build a JSON:API paginated document
// on top of an existing rest list response.
type Envelope struct {
	Total      int
	PageNumber int
	PageSize   int
}

// LastPage returns ceil(Total / PageSize), with a floor of 1.
func (e Envelope) LastPage() int {
	if e.PageSize <= 0 {
		return 1
	}
	if e.Total <= 0 {
		return 1
	}
	last := e.Total / e.PageSize
	if e.Total%e.PageSize != 0 {
		last++
	}
	return last
}

// Meta returns the meta block: { total, page: { number, size, last } }.
func (e Envelope) Meta() jsonapi.Meta {
	return jsonapi.Meta{
		"total": e.Total,
		"page": map[string]interface{}{
			"number": e.PageNumber,
			"size":   e.PageSize,
			"last":   e.LastPage(),
		},
	}
}

// BuildLinks returns the JSON:API top-level links for a paginated response.
//
// In normal range (1 <= current <= LastPage):
//   - self  -> page[number] = current             (always present)
//   - first -> page[number] = 1                   (always present)
//   - prev  -> page[number] = current - 1         (omitted on page 1)
//   - next  -> page[number] = current + 1         (omitted on last page)
//   - last  -> page[number] = LastPage            (always present)
//
// Past-end (current > LastPage):
//   - self  -> page[number] = current
//   - first -> page[number] = 1
//   - prev  -> page[number] = LastPage
//   - next  -> omitted
//   - last  -> page[number] = LastPage
func (e Envelope) BuildLinks(req *http.Request) jsonapi.Links {
	last := e.LastPage()
	links := jsonapi.Links{}

	links["self"] = jsonapi.Link{Href: rewritePage(req, e.PageNumber, e.PageSize)}
	links["first"] = jsonapi.Link{Href: rewritePage(req, 1, e.PageSize)}
	links["last"] = jsonapi.Link{Href: rewritePage(req, last, e.PageSize)}

	if e.PageNumber > last {
		// past-end: prev recovers to last, next omitted
		links["prev"] = jsonapi.Link{Href: rewritePage(req, last, e.PageSize)}
		return links
	}
	if e.PageNumber > 1 {
		links["prev"] = jsonapi.Link{Href: rewritePage(req, e.PageNumber-1, e.PageSize)}
	}
	if e.PageNumber < last {
		links["next"] = jsonapi.Link{Href: rewritePage(req, e.PageNumber+1, e.PageSize)}
	}
	return links
}

// rewritePage returns the request path with the query string preserved
// verbatim except that page[number] and page[size] are normalised to the
// supplied values. URL encoding follows net/url.Values.Encode() — emits
// %5B / %5D for [ / ], matching URLSearchParams on the client.
func rewritePage(req *http.Request, pageNumber, pageSize int) string {
	q := req.URL.Query()
	q.Set("page[number]", strconv.Itoa(pageNumber))
	q.Set("page[size]", strconv.Itoa(pageSize))
	encoded := url.Values(q).Encode()
	if encoded == "" {
		return req.URL.Path
	}
	return req.URL.Path + "?" + encoded
}
