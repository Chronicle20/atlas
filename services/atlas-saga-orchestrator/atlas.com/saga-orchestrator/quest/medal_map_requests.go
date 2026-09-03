package quest

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// medalMapBaseUrl names the atlas-quest domain for RootUrlFor's per-environment
// routing. It matches every other atlas-quest caller in the repo (e.g.
// quest/state/requests.go in this module), which use "QUESTS" and are
// overridable via QUESTS_SERVICE_URL.
const medalMapBaseUrl = "QUESTS"

func postMedalMap(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, questId uint32, mapId uint32) (medalMapRestModel, error) {
	return func(characterId uint32, questId uint32, mapId uint32) (medalMapRestModel, error) {
		root, err := requests.RootUrlFor(ctx, medalMapBaseUrl)
		if err != nil {
			return medalMapRestModel{}, err
		}
		url := fmt.Sprintf("%scharacters/%d/quests/%d/medal-maps", root, characterId, questId)
		body := medalMapPostRestModel{MapId: mapId}
		return requests.PostRequest[medalMapRestModel](url, body)(l, ctx)
	}
}
