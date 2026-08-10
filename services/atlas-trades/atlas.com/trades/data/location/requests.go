package location

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/location"
)

var baseURLProvider = func() string {
	return requests.RootUrl("MAPS")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestByCharacterId(characterId character.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("MAPS") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
