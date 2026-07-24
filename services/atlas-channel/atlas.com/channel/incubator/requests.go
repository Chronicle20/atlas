package incubator

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("GACHAPONS")
}

func dataBaseRequest() string {
	return requests.RootUrl("DATA")
}

// requestNpcById creates a request to atlas-data for one NPC template. A 404
// (mapped to requests.ErrNotFound by the request layer) means the NPC is absent
// from the tenant's game data.
func requestNpcById(npcId uint32) requests.Request[npcRestModel] {
	url := fmt.Sprintf("%sdata/npcs/%d", dataBaseRequest(), npcId)
	return requests.GetRequest[npcRestModel](url)
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
