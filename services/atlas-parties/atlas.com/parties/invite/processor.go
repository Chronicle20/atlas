package invite

import (
	"atlas-parties/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId byte, partyId uint32, targetId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId byte, partyId uint32, targetId uint32) error {
		return func(actorId uint32, worldId byte, partyId uint32, targetId uint32) error {
			l.Debugf("Creating party [%d] invitation for [%d] from [%d].", partyId, targetId, actorId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createInviteCommandProvider(actorId, partyId, worldId, targetId))
		}
	}
}
