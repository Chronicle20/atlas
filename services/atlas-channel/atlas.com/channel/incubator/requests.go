package incubator

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "GACHAPONS")
}

func dataBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// requestNpcById creates a request to atlas-data for one NPC template. A 404
// (mapped to requests.ErrNotFound by the request layer) means the NPC is absent
// from the tenant's game data.
func requestNpcById(ctx context.Context, npcId uint32) requests.Request[npcRestModel] {
	root, err := dataBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[npcRestModel](err)
	}
	url := fmt.Sprintf("%sdata/npcs/%d", root, npcId)
	return requests.GetRequest[npcRestModel](url)
}

// requestSelectReward creates a request to atlas-reward-pools that rolls one
// reward for the given gachapon (egg) id. The server reads no request body;
// a nil body is passed (mirrors atlas-saga-orchestrator's gachapon client)
// since jsonapi.Marshal would panic on a body value that does not implement
// MarshalIdentifier.
func requestSelectReward(ctx context.Context, eggId uint32) requests.Request[RewardRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RewardRestModel](err)
	}
	url := fmt.Sprintf("%sgachapons/%d/rewards/select", root, eggId)
	return requests.PostRequest[RewardRestModel](url, nil)
}
