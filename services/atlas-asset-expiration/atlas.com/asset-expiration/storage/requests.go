package storage

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "storage/accounts/%d?worldId=%d"
	Assets   = "storage/accounts/%d/assets?worldId=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "STORAGE")
}

// storageAssetsUrl returns the list URL for an account's storage assets. It
// is a bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func storageAssetsUrl(ctx context.Context, accountId uint32, worldId world.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Assets, accountId, worldId), nil
}
