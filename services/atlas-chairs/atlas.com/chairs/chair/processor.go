package chair

import (
	"atlas-chairs/kafka/producer"
	"atlas-chairs/map/data"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"math"
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
			l.Debugf("Attempting to sit in chair [%d] for character [%d].", chairId, characterId)
			_, err := GetById(ctx)(characterId)
			if err == nil {
				l.Errorf("Character [%d] already sitting on chair.", characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventErrorProvider(worldId, channelId, mapId, chairType, chairId, characterId, ErrorTypeAlreadySitting))
				return errors.New("already sitting")
			}

			if chairType == ChairTypeFixed {
				var m data.Model
				m, err = data.GetById(l)(ctx)(mapId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve map [%d] character [%d] is sitting in.", mapId, characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventErrorProvider(worldId, channelId, mapId, chairType, chairId, characterId, ErrorTypeInternal))
					return err
				}

				if chairId >= m.Seats() {
					l.Errorf("Character [%d] is attempting to sit in fixed chair [%d] in map [%d], but that does not exist.", characterId, chairId, mapId)
					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventErrorProvider(worldId, channelId, mapId, chairType, chairId, characterId, ErrorTypeDoesNotExist))
					return errors.New("chair does not exist")
				}

			}
			if chairType == ChairTypePortable {
				itemCategory := uint32(math.Floor(float64(chairId / 10000)))
				if itemCategory != 301 {
					l.Errorf("Character [%d] is attempting to sit in portable chair [%d] in map [%d], but that does not exist.", characterId, chairId, mapId)
					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventErrorProvider(worldId, channelId, mapId, chairType, chairId, characterId, ErrorTypeDoesNotExist))
					return errors.New("chair does not exist")
				}

				// TODO ensure character has item.
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
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(statusEventErrorProvider(worldId, channelId, mapId, chairType, chairId, characterId, ErrorTypeInternal))
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
