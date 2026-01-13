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
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func Login(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return func(ctx context.Context) func(transactionID uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		return func(transactionID uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.Debugf("Adding character [%d] from world [%d] to registry.", characterId, worldId)
				fm, err := getForeignCharacterInfo(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve needed character information from foreign service.")
					return err
				}
				c = CreateCharacter(ctx)(worldId, channelId, characterId, fm.Name())
			}

			l.Debugf("Setting character [%d] to online in registry.", characterId)
			fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
			c = UpdateCharacter(ctx)(c.Id(), Model.Login, fn)

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(character.EnvEventMemberStatusTopic)(loginEventProvider(transactionID, c.MessengerId(), c.WorldId(), characterId))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the messenger [%d] member [%d] logged in.", c.MessengerId(), c.Id())
					return err
				}
			}

			return nil
		}
	}
}

func Logout(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) error {
	return func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) error {
		return func(transactionID uuid.UUID, characterId uint32) error {
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to offline in registry.", characterId)
			c = UpdateCharacter(ctx)(c.Id(), Model.Logout)

			if c.MessengerId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(character.EnvEventMemberStatusTopic)(logoutEventProvider(transactionID, c.MessengerId(), c.WorldId(), characterId))
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
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in channel [%d] in registry.", characterId, channelId)
			fn := func(m Model) Model { return Model.ChangeChannel(m, channelId) }
			_ = UpdateCharacter(ctx)(c.Id(), fn)
			return nil
		}
	}
}

func JoinMessenger(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32, messengerId uint32) error {
	return func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32, messengerId uint32) error {
		return func(transactionID uuid.UUID, characterId uint32, messengerId uint32) error {
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to be in messenger [%d] in registry.", characterId, messengerId)
			fn := func(m Model) Model { return Model.JoinMessenger(m, messengerId) }
			_ = UpdateCharacter(ctx)(c.Id(), fn)
			return nil
		}
	}
}

func LeaveMessenger(l logrus.FieldLogger) func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) error {
	return func(ctx context.Context) func(transactionID uuid.UUID, characterId uint32) error {
		return func(transactionID uuid.UUID, characterId uint32) error {
			c, err := ByIdProvider(ctx)(characterId)()
			if err != nil {
				l.WithError(err).Warnf("Unable to locate character [%d] in registry.", characterId)
				return err
			}

			l.Debugf("Setting character [%d] to no longer have a messenger in the registry.", characterId)
			_ = UpdateCharacter(ctx)(c.Id(), Model.LeaveMessenger)
			return nil
		}
	}
}

func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
		return func(characterId uint32) model.Provider[Model] {
			return func() (Model, error) {
				c, err := ByIdProvider(ctx)(characterId)()
				if errors.Is(err, ErrNotFound) {
					fm, ferr := getForeignCharacterInfo(l)(ctx)(characterId)
					if ferr != nil {
						return Model{}, err
					}
					c = CreateCharacter(ctx)(fm.WorldId(), channel.Id(0), characterId, fm.Name())
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

// ============================================================================
// ProcessorImpl - struct-based processor pattern for consistency with other services
// ============================================================================

// ProcessorImpl provides struct-based processor methods for character operations.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new ProcessorImpl with the given logger and context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// Login processes a character login event.
func (p *ProcessorImpl) Login(transactionID uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return Login(p.l)(p.ctx)(transactionID, worldId, channelId, mapId, characterId)
}

// Logout processes a character logout event.
func (p *ProcessorImpl) Logout(transactionID uuid.UUID, characterId uint32) error {
	return Logout(p.l)(p.ctx)(transactionID, characterId)
}

// ChannelChange processes a character channel change event.
func (p *ProcessorImpl) ChannelChange(characterId uint32, channelId channel.Id) error {
	return ChannelChange(p.l)(p.ctx)(characterId, channelId)
}

// JoinMessenger updates the character to be in a messenger.
func (p *ProcessorImpl) JoinMessenger(transactionID uuid.UUID, characterId uint32, messengerId uint32) error {
	return JoinMessenger(p.l)(p.ctx)(transactionID, characterId, messengerId)
}

// LeaveMessenger updates the character to no longer be in a messenger.
func (p *ProcessorImpl) LeaveMessenger(transactionID uuid.UUID, characterId uint32) error {
	return LeaveMessenger(p.l)(p.ctx)(transactionID, characterId)
}

// GetById retrieves a character by ID, fetching from foreign service if not in local registry.
func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return GetById(p.l)(p.ctx)(characterId)
}
