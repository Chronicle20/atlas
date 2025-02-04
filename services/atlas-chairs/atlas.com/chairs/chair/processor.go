package chair

import (
	"atlas-chairs/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func GetById(ctx context.Context) func(characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	return func(characterId uint32) (Model, error) {
		return GetRegistry().Get(t, characterId)
	}
}

func Set(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) error {
			// TODO ensure chair is in map.
			// TODO ensure item is a chair.
			// TODO ensure character has item.

			l.Debugf("Attempting to sit in chair [%d] for character [%d].", chairId, characterId)
			_, err := GetById(ctx)(characterId)
			if err == nil {
				l.Errorf("Character [%d] already sitting on chair.", characterId)
				return errors.New("already sitting")
			}
			c := Model{
				worldId:   worldId,
				channelId: channelId,
				mapId:     mapId,
				id:        chairId,
				chairType: chairType,
			}

			_, err = GetRegistry().Set(t, characterId, c)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to sit on chair.", characterId)
				return err
			}
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventUsedProvider(worldId, channelId, mapId, chairType, chairId, characterId))
		}
	}
}

func Clear(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, mapId uint32) error {
	return func(ctx context.Context) func(characterId uint32, mapId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(characterId uint32, mapId uint32) error {
			l.Debugf("Attempting to clear for character [%d].", characterId)
			c, err := GetById(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Failed to get chair for character [%d].", characterId)
				return err
			}
			err = GetRegistry().Clear(t, characterId)
			if err != nil {
				l.WithError(err).Errorf("Failed to clear chair [%d] for character [%d].", c.Id(), characterId)
				return err
			}
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventCancelledProvider(c.WorldId(), c.ChannelId(), mapId, c.Type(), c.Id(), characterId))
		}
	}
}
