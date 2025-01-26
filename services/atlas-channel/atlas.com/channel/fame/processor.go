package fame

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func RequestChange(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestChangeFameCommandProvider(worldId, channelId, characterId, mapId, targetId, amount))
		}
	}
}
