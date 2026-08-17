package buff

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const characterBuffsResource = "characters/%d/buffs"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "BUFFS")
}

// characterBuffsUrl is a bare URL because atlas-buffs' list is paginated
// (task-117) and consumed via requests.DrainProvider.
func characterBuffsUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+characterBuffsResource, characterId), nil
}
