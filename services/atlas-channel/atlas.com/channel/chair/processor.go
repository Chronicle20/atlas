package chair

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Use(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
			l.Debugf("Character [%d] attempting to use map [%d] [%s] chair [%d].", characterId, mapId, chairType, chairId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(commandUseProvider(worldId, channelId, mapId, chairType, chairId, characterId))
		}
	}
}

func Cancel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(commandCancelProvider(worldId, channelId, mapId, characterId))
		}
	}
}
