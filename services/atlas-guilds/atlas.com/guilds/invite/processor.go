package invite

import (
	"atlas-guilds/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId byte, referenceId uint32, targetId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId byte, referenceId uint32, targetId uint32) error {
		return func(actorId uint32, worldId byte, referenceId uint32, targetId uint32) error {
			l.Debugf("Creating guild [%d] invitation for [%d] from [%d].", referenceId, targetId, actorId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createInviteCommandProvider(actorId, referenceId, worldId, targetId))
		}
	}
}
