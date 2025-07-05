package character

import (
	"atlas-messengers/kafka/message/character"
	"atlas-messengers/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Login(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		return func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.Debugf("Adding character [%d] from world [%d] to registry.", characterId, worldId)
				fm, err := getForeignCharacterInfo(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve needed character information from foreign service.")
					return err
				}
				c = GetRegistry().Create(t, worldId, channelId, characterId, fm.Name())
			}

			l.Debugf("Setting character [%d] to online in registry.", characterId)
			fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
			c = GetRegistry().Update(t, c.Id(), Model.Login, fn)

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(character.EnvEventMemberStatusTopic)(loginEventProvider(c.MessengerId(), c.WorldId(), characterId))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the messenger [%d] member [%d] logged in.", c.MessengerId(), c.Id())
					return err
				}
			}

			return nil
		}
	}
}

func Logout(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to offline in registry.", characterId)
			c = GetRegistry().Update(t, c.Id(), Model.Logout)

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(character.EnvEventMemberStatusTopic)(logoutEventProvider(c.MessengerId(), c.WorldId(), characterId))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the messenger [%d] member [%d] logged out.", c.MessengerId(), c.Id())
					return err
				}
			}

			return nil
		}
	}
}

func ChannelChange(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, channelId channel.Id) error {
	return func(ctx context.Context) func(characterId uint32, channelId channel.Id) error {
		return func(characterId uint32, channelId channel.Id) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in channel [%d] in registry.", characterId, channelId)
			fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
			c = GetRegistry().Update(t, c.Id(), fn)
			return nil
		}
	}
}

func JoinMessenger(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, messengerId uint32) error {
	return func(ctx context.Context) func(characterId uint32, messengerId uint32) error {
		return func(characterId uint32, messengerId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in messenger [%d] in registry.", characterId, messengerId)
			fn := func(m Model) Model { return Model.JoinMessenger(m, messengerId) }
			c = GetRegistry().Update(t, c.Id(), fn)
			return nil
		}
	}
}

func LeaveMessenger(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetRegistry().Get(t, characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to no longer have a messenger in the registry.", characterId)
			c = GetRegistry().Update(t, c.Id(), Model.LeaveMessenger)
			return nil
		}
	}
}

func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
		return func(characterId uint32) model.Provider[Model] {
			return func() (Model, error) {
				t := tenant.MustFromContext(ctx)
				c, err := GetRegistry().Get(t, characterId)
				if errors.Is(err, ErrNotFound) {
					fm, ferr := getForeignCharacterInfo(l)(ctx)(characterId)
					if ferr != nil {
						return Model{}, err
					}
					c = GetRegistry().Create(t, fm.WorldId(), channel.Id(0), characterId, fm.Name())
				}
				return c, nil
			}
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return byIdProvider(l)(ctx)(characterId)()
		}
	}
}

func getForeignCharacterInfo(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (ForeignModel, error) {
	return func(ctx context.Context) func(characterId uint32) (ForeignModel, error) {
		return func(characterId uint32) (ForeignModel, error) {
			return requests.Provider[ForeignRestModel, ForeignModel](l, ctx)(requestById(characterId), ExtractForeign)()
		}
	}
}
