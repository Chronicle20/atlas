package paginate

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ErrInvalidPageParam is returned for non-integer, out-of-range, or legacy
// paging parameters. Handlers map it to HTTP 400.
var ErrInvalidPageParam = errors.New("invalid page parameter")

// Repo-wide defaults (docs/rest-pagination.md). Group C game-capped lists
// pass MaxPageSize as their default so the common case fits one page.
const (
	DefaultPageSize = 50
	MaxPageSize     = 250
)

// ParseParams parses JSON:API page[number]/page[size] query params.
// Defaults: number=1, size=defaultSize. Invalid values (non-integer,
// number<1, size<1, size>maxSize) are an error, not silently clamped.
// The legacy ?limit= param is rejected outright, enforcing that paging is
// expressed only via page[*] repo-wide.
func ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error) {
	size := defaultSize
	if raw := query.Get("page[size]"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxSize {
			return model.Page{}, ErrInvalidPageParam
		}
		size = parsed
	}
	number := 1
	if raw := query.Get("page[number]"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return model.Page{}, ErrInvalidPageParam
		}
		number = parsed
	}
	if _, hasLimit := query["limit"]; hasLimit {
		return model.Page{}, ErrInvalidPageParam
	}
	return model.Page{Number: number, Size: size}, nil
}
