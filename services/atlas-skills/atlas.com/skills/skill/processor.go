package skill

import (
	"atlas-skills/database"
	"atlas-skills/kafka/message"
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

// Processor defines the interface for skill processing operations
type Processor interface {
	// ByCharacterIdProvider returns a provider for all skills for a character
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]

	// ByIdProvider returns a provider for a skill by ID
	ByIdProvider(characterId uint32, id uint32) model.Provider[Model]

	// Create creates a new skill with the given parameters
	Create(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error)

	// CreateAndEmit creates a new skill and emits a status event
	CreateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// Update updates an existing skill with the given parameters
	Update(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error)

	// UpdateAndEmit updates an existing skill and emits a status event
	UpdateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// SetCooldown applies a cooldown to a skill
	SetCooldown(mb *message.Buffer) func(characterId uint32) func(skillId uint32) func(cooldown uint32) (Model, error)

	// SetCooldownAndEmit applies a cooldown to a skill and emits a status event
	SetCooldownAndEmit(characterId uint32, skillId uint32, cooldown uint32) (Model, error)

	// ClearAll clears all cooldowns for a character
	ClearAll(characterId uint32) error

	// Delete deletes all skills for a character
	Delete(characterId uint32) error

	CooldownDecorator(characterId uint32) model.Decorator[Model]

	RequestCreate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error

	RequestUpdate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

// NewProcessor creates a new ProcessorImpl
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}

// ByCharacterIdProvider returns a provider for all skills for a character
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	mp := model.SliceMap(Make)(getByCharacterId(p.t.Id(), characterId)(p.db))()
	return model.SliceMap(model.Decorate(model.Decorators(p.CooldownDecorator(characterId))))(mp)(model.ParallelMap())
}

// ByIdProvider returns a provider for a skill by ID
func (p *ProcessorImpl) ByIdProvider(characterId uint32, id uint32) model.Provider[Model] {
	mp := model.Map(Make)(getById(p.t.Id(), characterId, id)(p.db))
	return model.Map(model.Decorate(model.Decorators(p.CooldownDecorator(characterId))))(mp)
}

// Create creates a new skill with the given parameters
func (p *ProcessorImpl) Create(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
	return func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
		return func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
			return func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
				return func(masterLevel byte) func(expiration time.Time) (Model, error) {
					return func(expiration time.Time) (Model, error) {
						p.l.Debugf("Attempting to create skill [%d] for character [%d].", id, characterId)
						var s Model
						txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
							var err error
							s, err = p.WithTransaction(tx).ByIdProvider(characterId, id)()
							if s.Id() != 0 {
								return errors.New("already exists")
							}
							s, err = create(tx, p.t.Id(), characterId, id, level, masterLevel, expiration)
							if err != nil {
								return err
							}
							return nil
						})
						if txErr != nil {
							return Model{}, txErr
						}
						p.l.Debugf("Created skill [%d] for character [%d].", id, characterId)

						// Add the status event to the message buffer
						_ = mb.Put(skill2.EnvStatusEventTopic, statusEventCreatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))

						return s, nil
					}
				}
			}
		}
	}
}

// CreateAndEmit creates a new skill and emits a status event
func (p *ProcessorImpl) CreateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	var s Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		s, err = p.Create(buf)(characterId)(id)(level)(masterLevel)(expiration)
		return err
	})
	return s, err
}

// Update updates an existing skill with the given parameters
func (p *ProcessorImpl) Update(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
	return func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
		return func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
			return func(level byte) func(masterLevel byte) func(expiration time.Time) (Model, error) {
				return func(masterLevel byte) func(expiration time.Time) (Model, error) {
					return func(expiration time.Time) (Model, error) {
						p.l.Debugf("Attempting to update skill [%d] for character [%d].", id, characterId)
						var s Model
						txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
							var err error
							s, err = p.WithTransaction(tx).ByIdProvider(characterId, id)()
							if err != nil {
								return errors.New("does not exist")
							}
							err = dynamicUpdate(tx)(SetLevel(level), SetMasterLevel(masterLevel), SetExpiration(expiration))(p.t.Id(), characterId)(s)
							if err != nil {
								return err
							}
							s, err = p.WithTransaction(tx).ByIdProvider(characterId, id)()
							if err != nil {
								return errors.New("does not exist")
							}
							return nil
						})
						if txErr != nil {
							return Model{}, txErr
						}
						p.l.Debugf("Update skill [%d] for character [%d].", id, characterId)

						// Add the status event to the message buffer
						_ = mb.Put(skill2.EnvStatusEventTopic, statusEventUpdatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))

						return s, nil
					}
				}
			}
		}
	}
}

// UpdateAndEmit updates an existing skill and emits a status event
func (p *ProcessorImpl) UpdateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	var s Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		s, err = p.Update(buf)(characterId)(id)(level)(masterLevel)(expiration)
		return err
	})
	return s, err
}

// SetCooldown applies a cooldown to a skill
func (p *ProcessorImpl) SetCooldown(mb *message.Buffer) func(characterId uint32) func(skillId uint32) func(cooldown uint32) (Model, error) {
	return func(characterId uint32) func(skillId uint32) func(cooldown uint32) (Model, error) {
		return func(skillId uint32) func(cooldown uint32) (Model, error) {
			return func(cooldown uint32) (Model, error) {
				p.l.Debugf("Applying cooldown of [%d] for character [%d] skill [%d].", cooldown, characterId, skillId)
				err := GetRegistry().Apply(p.t, characterId, skillId, cooldown)
				if err != nil {
					return Model{}, err
				}
				s, err := p.ByIdProvider(characterId, skillId)()
				if err != nil {
					return Model{}, err
				}

				// Add the status event to the message buffer
				_ = mb.Put(skill2.EnvStatusEventTopic, statusEventCooldownAppliedProvider(characterId, s.Id(), s.CooldownExpiresAt()))

				return s, nil
			}
		}
	}
}

// SetCooldownAndEmit applies a cooldown to a skill and emits a status event
func (p *ProcessorImpl) SetCooldownAndEmit(characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
	var s Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		s, err = p.SetCooldown(buf)(characterId)(skillId)(cooldown)
		return err
	})
	return s, err
}

// ExpireCooldowns expires all cooldowns that have passed their expiration time
func ExpireCooldowns(l logrus.FieldLogger, ctx context.Context) {
	for _, s := range GetRegistry().GetAll() {
		if s.CooldownExpiresAt().Before(time.Now()) {
			tctx := tenant.WithContext(ctx, s.Tenant())
			_ = GetRegistry().Clear(s.Tenant(), s.CharacterId(), s.SkillId())
			_ = producer.ProviderImpl(l)(tctx)(skill2.EnvStatusEventTopic)(statusEventCooldownExpiredProvider(s.CharacterId(), s.SkillId()))
		}
	}
}

// ClearAll clears all cooldowns for a character
func (p *ProcessorImpl) ClearAll(characterId uint32) error {
	return GetRegistry().ClearAll(p.t, characterId)
}

// Delete deletes all skills for a character
func (p *ProcessorImpl) Delete(characterId uint32) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return deleteByCharacter(tx, p.t, characterId)
	})
}

// CooldownDecorator returns a decorator that adds cooldown information to a skill model
func (p *ProcessorImpl) CooldownDecorator(characterId uint32) model.Decorator[Model] {
	return func(m Model) Model {
		ct, err := GetRegistry().Get(p.t, characterId, m.Id())
		if err != nil {
			return m
		}
		return m.SetCooldown(ct)
	}
}

// RequestCreate sends a command to create a skill
func (p *ProcessorImpl) RequestCreate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(createCommandProvider(characterId, id, level, masterLevel, expiration))
}

// RequestUpdate sends a command to update a skill
func (p *ProcessorImpl) RequestUpdate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(updateCommandProvider(characterId, id, level, masterLevel, expiration))
}
