package incubator

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("GACHAPONS")
}

// requestSelectReward creates a request to atlas-reward-pools that rolls one
// reward for the given gachapon (egg) id. The server reads no request body;
// a nil body is passed (mirrors atlas-saga-orchestrator's gachapon client)
// since jsonapi.Marshal would panic on a body value that does not implement
// MarshalIdentifier.
func requestSelectReward(eggId uint32) requests.Request[RewardRestModel] {
	url := fmt.Sprintf("%sgachapons/%d/rewards/select", getBaseRequest(), eggId)
	return requests.PostRequest[RewardRestModel](url, nil)
}
