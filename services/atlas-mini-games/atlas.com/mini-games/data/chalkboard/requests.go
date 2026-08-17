package chalkboard

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ById = "chalkboards/%d"
)

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHALKBOARDS")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestById(ctx context.Context, characterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, characterId))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrlFor("CHALKBOARDS")
// default. The injected closure ignores ctx -- tests always exercise the
// fixed httptest URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
