package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	questsResource    = "data/quests"
	questByIdResource = "data/quests/%d"
	autoStartQuests   = "data/quests/auto-start"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, questId uint32) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+questByIdResource, questId))
}

// allQuestsUrl and autoStartQuestsUrl are bare URLs (not requests.Request)
// because both lists are now paginated server-side (task-117) and consumed
// via requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func allQuestsUrl(ctx context.Context) string  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + questsResource, nil
}

func autoStartQuestsUrl(ctx context.Context) string  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + autoStartQuests, nil
}
