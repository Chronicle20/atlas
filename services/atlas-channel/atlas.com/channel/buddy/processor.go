package buddy

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func RequestAdd(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, targetId uint32, group string) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, targetId uint32, group string) error {
		return func(characterId uint32, worldId byte, targetId uint32, group string) error {
			l.Debugf("Character [%d] would like to add [%d] to group [%s] to their buddy list.", characterId, targetId, group)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestAddBuddyCommandProvider(characterId, worldId, targetId, group))
		}
	}
}

func RequestDelete(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, targetId uint32) error {
		return func(characterId uint32, worldId byte, targetId uint32) error {
			l.Debugf("Character [%d] attempting to delete buddy [%d].", characterId, targetId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestDeleteBuddyCommandProvider(characterId, worldId, targetId))
		}
	}
}
