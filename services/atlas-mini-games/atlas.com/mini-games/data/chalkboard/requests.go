package chalkboard

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ById = "chalkboards/%d"
)

var baseURLProvider = func() string {
	return requests.RootUrl("CHALKBOARDS")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestById(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, characterId))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("CHALKBOARDS") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
