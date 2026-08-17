package rewardpool

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "GACHAPONS")
}

// requestSelectReward creates a request to atlas-reward-pools that rolls one
// reward for the given box template id. A cash-surprise pool's id is the box
// template id (design.md §4.1), so this already resolves the right pool with
// no new endpoint. The server reads no request body; a nil body is passed
// since jsonapi.Marshal would panic on a body value that does not implement
// MarshalIdentifier.
func requestSelectReward(ctx context.Context, boxTemplateId uint32) requests.Request[RewardRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RewardRestModel](err)
	}
	url := fmt.Sprintf("%sgachapons/%d/rewards/select", root, boxTemplateId)
	return requests.PostRequest[RewardRestModel](url, nil)
}
