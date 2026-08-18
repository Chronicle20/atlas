package minigame

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "worlds/%d/channels/%d/maps/%d/instances/%s/games"
	CharacterResource = "characters/%d/games"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MINI_GAMES")
}

func requestInField(ctx context.Context, f field.Model) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

// requestByMember reads the (0-or-1) mini-game room characterId is seated in.
func requestByMember(ctx context.Context, characterId uint32) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+CharacterResource, characterId))
}
