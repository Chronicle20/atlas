package quest

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// medalMapBaseUrl names the atlas-quest domain for RootUrlFor's per-environment
// routing, following the CHARACTER_URL/GACHAPONS_URL/RPS_URL convention
// (services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/*/requests.go).
// atlas-quest has no other REST-client caller in this service yet.
const medalMapBaseUrl = "QUEST_URL"

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
