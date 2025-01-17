package invite

import (
	"atlas-invites/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const StartInviteId = uint32(1000000000)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) error {
	return func(ctx context.Context) func(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) error {
		return func(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) error {
			t := tenant.MustFromContext(ctx)
			i := GetRegistry().Create(t, originatorId, worldId, targetId, inviteType, referenceId)
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
		}
	}
}

func Accept(l logrus.FieldLogger) func(ctx context.Context) func(referenceId uint32, worldId byte, inviteType string, actorId uint32) error {
	return func(ctx context.Context) func(referenceId uint32, worldId byte, inviteType string, actorId uint32) error {
		return func(referenceId uint32, worldId byte, inviteType string, actorId uint32) error {
			t := tenant.MustFromContext(ctx)
			i, err := GetRegistry().GetByReference(t, actorId, inviteType, referenceId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate invite being acted upon.")
				return err
			}

			err = GetRegistry().Delete(t, actorId, inviteType, i.OriginatorId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate invite being acted upon.")
				return err
			}

			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(acceptedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
		}
	}
}

func Reject(l logrus.FieldLogger) func(ctx context.Context) func(originatorId uint32, worldId byte, inviteType string, actorId uint32) error {
	return func(ctx context.Context) func(originatorId uint32, worldId byte, inviteType string, actorId uint32) error {
		return func(originatorId uint32, worldId byte, inviteType string, actorId uint32) error {
			t := tenant.MustFromContext(ctx)
			i, err := GetRegistry().GetByOriginator(t, actorId, inviteType, originatorId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate invite being acted upon.")
				return err
			}

			err = GetRegistry().Delete(t, actorId, inviteType, originatorId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate invite being acted upon.")
				return err
			}

			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(rejectedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
		}
	}
}

func byCharacterIdProvider(_ logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			t := tenant.MustFromContext(ctx)
			is, err := GetRegistry().GetForCharacter(t, characterId)
			if err != nil {
				return model.ErrorProvider[[]Model](err)
			}
			return model.FixedProvider(is)
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(l)(ctx)(characterId)()
		}
	}
}
