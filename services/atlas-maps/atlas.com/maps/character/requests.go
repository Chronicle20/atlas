package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

// baseURLProvider is the seam used by tests to redirect requests to an
// httptest server. Production code uses requests.RootUrl("CHARACTERS").
var baseURLProvider = func() string { return requests.RootUrl("CHARACTERS") }

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(baseURLProvider()+ById, id))
}
