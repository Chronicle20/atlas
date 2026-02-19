package party_quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	TimerByCharacterId = "party-quests/instances/character/%d/timer"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTY_QUESTS")
}

func requestTimerByCharacterId(characterId uint32) requests.Request[TimerRestModel] {
	return requests.GetRequest[TimerRestModel](fmt.Sprintf(getBaseRequest()+TimerByCharacterId, characterId))
}
