package rps

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "RPS_URL"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, BaseUrl)
}

func requestStartGame(ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) requests.Request[RestModel] {
	body := RestModel{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		NpcId:       npcId,
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](
		fmt.Sprintf("%srps/games", root), body)
}

// StartGame opens (or re-opens) an RPS session for a character at the given
// NPC, by POSTing to atlas-rps's synchronous /rps/games endpoint.
func StartGame(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (RestModel, error) {
	return func(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (RestModel, error) {
		return requestStartGame(ctx, characterId, worldId, channelId, npcId)(l, ctx)
	}
}
