package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

// allCharactersUrl is a bare URL (not a requests.Request) because
// GET /characters is paginated server-side (paginate.DefaultPageSize=50,
// see services/atlas-character/atlas.com/character/character/resource.go)
// and is consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func allCharactersUrl() string {
	return getBaseRequest() + Resource
}
