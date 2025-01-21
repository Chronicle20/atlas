package guild

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
			l.Debugf("Character [%d] attempting to create guild [%s] in world [%d] channel [%d] map [%d].", characterId, name, worldId, channelId, mapId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestCreateProvider(worldId, channelId, mapId, characterId, name))
		}
	}
}
