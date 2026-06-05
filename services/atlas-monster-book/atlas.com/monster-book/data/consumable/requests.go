package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the atlas-data consumable collection path.
	Resource = "data/consumables"
	// ById is the single-consumable path template.
	ById = Resource + "/%d"
)

var baseURLProvider = func() string {
	return requests.RootUrl("DATA")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("DATA") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
