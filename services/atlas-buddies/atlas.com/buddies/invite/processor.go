package invite

import (
	"atlas-buddies/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId byte, targetId uint32) error {
		return func(actorId uint32, worldId byte, targetId uint32) error {
			l.Debugf("Creating buddy [%d] invitation for [%d].", targetId, actorId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createInviteCommandProvider(actorId, worldId, targetId))
		}
	}
}

func Reject(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId byte, originatorId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId byte, originatorId uint32) error {
		return func(actorId uint32, worldId byte, originatorId uint32) error {
			l.Debugf("Rejecting buddy [%d] invitation for [%d].", originatorId, actorId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(rejectInviteCommandProvider(actorId, worldId, originatorId))
		}
	}
}
