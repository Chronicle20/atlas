package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	questPath           = "data/quests/%d"
	autoStartQuestsPath = "data/quests/auto-start"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestQuestById(ctx context.Context, questId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+questPath, questId))
}

// autoStartQuestsUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func autoStartQuestsUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + autoStartQuestsPath, nil
}
