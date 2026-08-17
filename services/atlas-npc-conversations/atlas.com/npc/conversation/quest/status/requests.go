package status

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUEST")
}

// RequestByCharacterAndQuest returns a request to get quest status for a character
// Calls GET /characters/{characterId}/quests/{questId}
func RequestByCharacterAndQuest(ctx context.Context, characterId uint32, questId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+"characters/%d/quests/%d", characterId, questId))
}
