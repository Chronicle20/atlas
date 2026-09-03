package progress

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUEST")
}

// ByCharacterAndQuestUrl returns the bare URL for a character's quest
// progress collection. It is a bare URL (not a requests.Request) because
// the collection is paginated server-side and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func ByCharacterAndQuestUrl(ctx context.Context, characterId uint32, questId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+"characters/%d/quests/%d/progress", characterId, questId), nil
}
