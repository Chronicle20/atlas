package invite

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func Accept(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
		return func(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(acceptInviteCommandProvider(actorId, worldId, inviteType, referenceId))
		}
	}
}

func Reject(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
	return func(ctx context.Context) func(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
		return func(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(rejectInviteCommandProvider(actorId, worldId, inviteType, originatorId))
		}
	}
}
