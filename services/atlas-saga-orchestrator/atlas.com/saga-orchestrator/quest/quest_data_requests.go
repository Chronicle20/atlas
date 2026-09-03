package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// questDataBaseUrl names the atlas-data domain, matching every other
// atlas-data caller in this module (e.g. data/foothold/requests.go), which
// use "DATA" and are overridable via DATA_SERVICE_URL.
const questDataBaseUrl = "DATA"

const questDataPath = "data/quests/%d"

func requestQuestData(ctx context.Context, questId uint32) requests.Request[questDataRestModel] {
	root, err := requests.RootUrlFor(ctx, questDataBaseUrl)
	if err != nil {
		return requests.ErrorRequest[questDataRestModel](err)
	}
	return requests.GetRequest[questDataRestModel](fmt.Sprintf(root+questDataPath, questId))
}
