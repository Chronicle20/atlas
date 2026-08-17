package worldbroadcast

import (
	"context"
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

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "WORLDS")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestQueue(ctx context.Context, worldId world.Id, family string) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, worldId, family))
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only call
// from a test; production code uses the env-driven default (mirrors
// monsterbook/requests.go's SetBaseURLForTest). The injected closure
// ignores ctx -- tests always exercise the fixed httptest URL regardless
// of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
