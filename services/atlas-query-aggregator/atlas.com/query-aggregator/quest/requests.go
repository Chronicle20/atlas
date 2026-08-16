package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	CharactersResource = "characters"
	QuestsResource     = "quests"
	ByCharacterAndId   = CharactersResource + "/%d/" + QuestsResource + "/%d"
	ByCharacter        = CharactersResource + "/%d/" + QuestsResource
	StartedQuests      = CharactersResource + "/%d/" + QuestsResource + "/started"
	CompletedQuests    = CharactersResource + "/%d/" + QuestsResource + "/completed"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUESTS")
}

func requestById(ctx context.Context, characterId uint32, questId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ByCharacterAndId, characterId, questId))
}

// byCharacterUrl returns the list URL for a character's quests. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func byCharacterUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+ByCharacter, characterId), nil
}

// startedByCharacterUrl and completedByCharacterUrl are bare URLs (not
// requests.Request) because both lists are now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func startedByCharacterUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+StartedQuests, characterId), nil
}

func completedByCharacterUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+CompletedQuests, characterId), nil
}
