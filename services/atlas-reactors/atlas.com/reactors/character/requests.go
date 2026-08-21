package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

// baseURLProvider is the seam used by tests to redirect requests to an
// httptest server. Production code resolves the caller's environment via
// requests.RootUrlFor("CHARACTERS").
var baseURLProvider = func(ctx context.Context) (string, error) { return requests.RootUrlFor(ctx, "CHARACTERS") }

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := baseURLProvider(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}
