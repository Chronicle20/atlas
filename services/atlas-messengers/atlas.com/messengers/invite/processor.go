package invite

import (
	"atlas-messengers/kafka/message/invite"
	"atlas-messengers/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId byte, messengerId uint32, targetId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId byte, messengerId uint32, targetId uint32) error {
		return func(actorId uint32, worldId byte, messengerId uint32, targetId uint32) error {
			l.Debugf("Creating messenger [%d] invitation for [%d] from [%d].", messengerId, targetId, actorId)
			return producer.ProviderImpl(l)(ctx)(invite.EnvCommandTopic)(createInviteCommandProvider(actorId, messengerId, worldId, targetId))
		}
	}
}
