package worldbroadcast

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the path template for fetching one (worldId, family)
	// broadcast queue from atlas-world. Must match the route atlas-world
	// registered in Task 9: /worlds/{worldId}/broadcast-queues/{family}
	// (services/atlas-world/atlas.com/world/broadcast/resource.go).
	Resource = "worlds/%d/broadcast-queues/%s"
)

var baseURLProvider = func() string {
	return requests.RootUrl("WORLDS")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestQueue(worldId world.Id, family string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId, family))
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only call
// from a test; production code uses the env-driven default (mirrors
// monsterbook/requests.go's SetBaseURLForTest).
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
