package character

import (
	"atlas-parties/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Login(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.Debugf("Adding character [%d] from world [%d] to registry.", characterId, worldId)
				fm, err := getForeignCharacterInfo(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve needed character information from foreign service.")
					return err
				}
				c = GetRegistry().Create(t, worldId, channelId, mapId, characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
			}

			l.Debugf("Setting character [%d] to online in registry.", characterId)
			fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
			c = GetRegistry().Update(t, c.Id(), Model.Login, fn)

			if c.PartyId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(EnvEventMemberStatusTopic)(loginEventProvider(c.PartyId(), c.WorldId(), characterId))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] logged in.", c.PartyId(), c.Id())
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

			if c.PartyId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(EnvEventMemberStatusTopic)(logoutEventProvider(c.PartyId(), c.WorldId(), characterId))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] logged out.", c.PartyId(), c.Id())
					return err
				}
			}

			return nil
		}
	}
}

func ChannelChange(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, channelId byte) error {
	return func(ctx context.Context) func(characterId uint32, channelId byte) error {
		return func(characterId uint32, channelId byte) error {
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

func LevelChange(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			// TODO
			return nil
		}
	}
}

func JobChange(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			// TODO
			return nil
		}
	}
}

func MapChange(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, mapId uint32) error {
	return func(ctx context.Context) func(characterId uint32, mapId uint32) error {
		return func(characterId uint32, mapId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in map [%d] in registry.", characterId, mapId)
			fn := func(m Model) Model { return Model.ChangeMap(m, mapId) }
			c = GetRegistry().Update(t, c.Id(), fn)
			return nil
		}
	}
}

func JoinParty(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, partyId uint32) error {
	return func(ctx context.Context) func(characterId uint32, partyId uint32) error {
		return func(characterId uint32, partyId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in party [%d] in registry.", characterId, partyId)
			fn := func(m Model) Model { return Model.JoinParty(m, partyId) }
			c = GetRegistry().Update(t, c.Id(), fn)
			return nil
		}
	}
}

func LeaveParty(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetRegistry().Get(t, characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to no longer have a party in the registry.", characterId)
			c = GetRegistry().Update(t, c.Id(), Model.LeaveParty)
			return nil
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			t := tenant.MustFromContext(ctx)
			c, err := GetRegistry().Get(t, characterId)
			if err != nil {
				l.Warnf("Character [%d] not found in registry, may have already been deleted.", characterId)
				return nil
			}

			if c.PartyId() != 0 {
				l.Debugf("Character [%d] was in party [%d], leaving party before deletion.", characterId, c.PartyId())
				err = LeaveParty(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to remove character [%d] from party [%d] before deletion.", characterId, c.PartyId())
					return err
				}
			}

			l.Debugf("Removing character [%d] from registry.", characterId)
			err = GetRegistry().Delete(t, characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to delete character [%d] from registry.", characterId)
				return err
			}

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
					c = GetRegistry().Create(t, fm.WorldId(), 0, fm.MapId(), characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
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
