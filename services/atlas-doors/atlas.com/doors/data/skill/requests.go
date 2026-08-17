package skill

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	skillsResource = "data/skills/%d"
)

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, skillId skill.Id) requests.Request[RestModel] {
	root, err := baseURLProvider(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+skillsResource, uint32(skillId)))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrlFor("DATA") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
