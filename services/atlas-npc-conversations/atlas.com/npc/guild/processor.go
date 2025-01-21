package guild

import (
	"atlas-npc-conversations/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func IsLeader(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			//TODO
			return false
		}
	}
}

func HasGuild(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			//TODO
			return false
		}
	}
}

func RequestName(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			l.Debugf("Requesting character [%d] input guild name for creation.", characterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestNameProvider(worldId, channelId, characterId))
		}
	}
}
