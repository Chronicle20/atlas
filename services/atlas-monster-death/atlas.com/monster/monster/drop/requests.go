package drop

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	MonsterDropsResource = "monsters/%d/drops"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DROPS_INFORMATION")
}

// monsterDropsUrl is a bare URL (not a requests.Request) because the list is
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func monsterDropsUrl(ctx context.Context, monsterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+MonsterDropsResource, monsterId), nil
}
