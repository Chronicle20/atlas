package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	StartedQuestsResource = "characters/%d/quests/started"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUESTS")
}

// startedQuestsUrl returns the list URL for a character's started quests.
// It is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func startedQuestsUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+StartedQuestsResource, characterId), nil
}
