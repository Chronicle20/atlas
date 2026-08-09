package rewardpool

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("GACHAPONS")
}

// requestSelectReward creates a request to atlas-reward-pools that rolls one
// reward for the given box template id. A cash-surprise pool's id is the box
// template id (design.md §4.1), so this already resolves the right pool with
// no new endpoint. The server reads no request body; a nil body is passed
// since jsonapi.Marshal would panic on a body value that does not implement
// MarshalIdentifier.
func requestSelectReward(boxTemplateId uint32) requests.Request[RewardRestModel] {
	url := fmt.Sprintf("%sgachapons/%d/rewards/select", getBaseRequest(), boxTemplateId)
	return requests.PostRequest[RewardRestModel](url, nil)
}
