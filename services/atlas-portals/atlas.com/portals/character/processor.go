package character

import (
	"atlas-portals/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func EnableActions(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
		return func(worldId byte, channelId byte, characterId uint32) {
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(enableActionsProvider(worldId, channelId, characterId))
		}
	}
}
