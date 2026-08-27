package ring

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "rings"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

// requestByCharacterId returns the list URL for a character's ring pairs
// (atlas-cashshop/ring/resource.go:29, filter[characterId] required). Bare
// URL (not a requests.Request) because the list is paginated server-side
// (task-269 task 8, mirroring task-117's door list) and must be consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request -- see door/requests.go:20-32.
func requestByCharacterId(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource+"?filter[characterId]=%d", characterId), nil
}
