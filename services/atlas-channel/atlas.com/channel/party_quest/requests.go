package party_quest

import (
	"atlas-channel/rest"
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
	return rest.MakeGetRequest[TimerRestModel](fmt.Sprintf(getBaseRequest()+TimerByCharacterId, characterId))
}
