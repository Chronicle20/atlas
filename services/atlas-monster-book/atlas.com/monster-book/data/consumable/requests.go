package consumable

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the atlas-data consumable collection path.
	Resource = "data/consumables"
	// ById is the single-consumable path template.
	ById = Resource + "/%d"
)

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrlFor("DATA") default.
// The injected closure ignores ctx — tests always exercise the fixed
// httptest URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
