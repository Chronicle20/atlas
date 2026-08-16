package party_quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	TimerByCharacterId = "party-quests/instances/character/%d/timer"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PARTY_QUESTS")
}

func requestTimerByCharacterId(ctx context.Context, characterId uint32) requests.Request[TimerRestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[TimerRestModel](err)
	}
	return requests.GetRequest[TimerRestModel](fmt.Sprintf(root+TimerByCharacterId, characterId))
}
