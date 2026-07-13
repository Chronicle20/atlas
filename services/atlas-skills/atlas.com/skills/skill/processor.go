package skill

import (
	"atlas-skills/kafka/message"
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/macro"
	"context"
	"errors"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	constskill "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor defines the interface for skill processing operations
type Processor interface {
	// ByCharacterIdProvider returns a provider for all skills for a character
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]

	// ByIdProvider returns a provider for a skill by ID
	ByIdProvider(characterId uint32, id uint32) model.Provider[Model]

	// Create creates a new skill with the given parameters
	Create(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// CreateAndEmit creates a new skill and emits a status event
	CreateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// Update updates an existing skill with the given parameters
	Update(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// UpdateAndEmit updates an existing skill and emits a status event
	UpdateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error)

	// SetCooldown applies a cooldown to a skill
	SetCooldown(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, cooldown uint32) (Model, error)

	// SetCooldownAndEmit applies a cooldown to a skill and emits a status event
	SetCooldownAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, cooldown uint32) (Model, error)

	// ClearAll clears all cooldowns for a character
	ClearAll(characterId uint32) error

	// Delete deletes all skills for a character
	Delete(characterId uint32) error

	CooldownDecorator(characterId uint32) model.Decorator[Model]

	RequestCreate(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error

	RequestUpdate(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error

	// DeleteForSagaCompensationAndEmit deletes a skill for a saga compensation step.
	// Idempotent on missing rows — an absent skill still emits StatusEventTypeDeleted
	// so the orchestrator's correlator treats it as success (plan Phase 5).
	DeleteForSagaCompensationAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error

	// DeleteForSagaCompensation is the buffer-based inner form (tests use this to
	// avoid Kafka dependency).
	DeleteForSagaCompensation(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error

	// WithTransaction returns a Processor that executes against the given transaction
	WithTransaction(tx *gorm.DB) Processor

	// TransferSp moves one skill point from fromSkillId to toSkillId (SP Reset
	// item 505000<itemTier>), re-validating job tree, exclusion list, tier, and
	// level/cap state authoritatively, and cleaning up macro references when the
	// source drops to level 0. All mutation happens inside a single gorm
	// transaction (see TransferSp's implementation) — see task-126 design §4.6.
	TransferSp(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error

	// TransferSpAndEmit wraps TransferSp with the producer emit flow.
	TransferSpAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error
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

var _ Processor = (*ProcessorImpl)(nil)

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
	mp := model.SliceMap(Make)(getByCharacterId(characterId)(p.db.WithContext(p.ctx)))()
	return model.SliceMap(model.Decorate(model.Decorators(p.CooldownDecorator(characterId))))(mp)(model.ParallelMap())
}

// ByIdProvider returns a provider for a skill by ID
func (p *ProcessorImpl) ByIdProvider(characterId uint32, id uint32) model.Provider[Model] {
	mp := model.Map(Make)(getById(characterId, id)(p.db.WithContext(p.ctx)))
	return model.Map(model.Decorate(model.Decorators(p.CooldownDecorator(characterId))))(mp)
}

// Create creates a new skill with the given parameters
func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		p.l.Debugf("Attempting to create skill [%d] for character [%d].", id, characterId)
		var s Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
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
		_ = mb.Put(skill2.EnvStatusEventTopic, statusEventCreatedProvider(transactionId, worldId, characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))

		return s, nil
	}
}

// CreateAndEmit creates a new skill and emits a status event. The create
// write and the outbox enqueue share one transaction (Create's own
// ExecuteTransaction is safely re-entrant inside this outer one).
func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	var s Model
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			s, err = p.WithTransaction(tx).Create(buf)(transactionId, worldId, characterId, id, level, masterLevel, expiration)
			return err
		})
	})
	return s, err
}

// Update updates an existing skill with the given parameters
func (p *ProcessorImpl) Update(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		p.l.Debugf("Attempting to update skill [%d] for character [%d].", id, characterId)
		var s Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			var err error
			s, err = p.WithTransaction(tx).ByIdProvider(characterId, id)()
			if err != nil {
				return errors.New("does not exist")
			}
			err = dynamicUpdate(tx)(SetLevel(level), SetMasterLevel(masterLevel), SetExpiration(expiration))(characterId)(s)
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
		_ = mb.Put(skill2.EnvStatusEventTopic, statusEventUpdatedProvider(transactionId, worldId, characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))

		return s, nil
	}
}

// UpdateAndEmit updates an existing skill and emits a status event. The
// update write and the outbox enqueue share one transaction (Update's own
// ExecuteTransaction is safely re-entrant inside this outer one).
func (p *ProcessorImpl) UpdateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	var s Model
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			s, err = p.WithTransaction(tx).Update(buf)(transactionId, worldId, characterId, id, level, masterLevel, expiration)
			return err
		})
	})
	return s, err
}

// SetCooldown applies a cooldown to a skill
func (p *ProcessorImpl) SetCooldown(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
		p.l.Debugf("Applying cooldown of [%d] for character [%d] skill [%d].", cooldown, characterId, skillId)
		err := GetRegistry().Apply(p.ctx, characterId, skillId, cooldown)
		if err != nil {
			return Model{}, err
		}
		s, err := p.ByIdProvider(characterId, skillId)()
		if err != nil {
			return Model{}, err
		}

		// Add the status event to the message buffer
		_ = mb.Put(skill2.EnvStatusEventTopic, statusEventCooldownAppliedProvider(transactionId, worldId, characterId, s.Id(), s.CooldownExpiresAt()))

		return s, nil
	}
}

// SetCooldownAndEmit applies a cooldown to a skill and emits a status event
func (p *ProcessorImpl) SetCooldownAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
	var s Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		s, err = p.SetCooldown(buf)(transactionId, worldId, characterId, skillId, cooldown)
		return err
	})
	return s, err
}

// ExpireCooldowns expires all cooldowns that have passed their expiration time
func ExpireCooldowns(l logrus.FieldLogger, ctx context.Context) {
	for _, s := range GetRegistry().GetAll(ctx) {
		if s.CooldownExpiresAt().Before(time.Now()) {
			tctx := tenant.WithContext(ctx, s.Tenant())
			_ = GetRegistry().Clear(tctx, s.CharacterId(), s.SkillId())
			// Use zero values for transactionId and worldId since this is a background expiration
			_ = producer.ProviderImpl(l)(tctx)(skill2.EnvStatusEventTopic)(statusEventCooldownExpiredProvider(uuid.Nil, world.Id(0), s.CharacterId(), s.SkillId()))
		}
	}
}

// ClearAll clears all cooldowns for a character
func (p *ProcessorImpl) ClearAll(characterId uint32) error {
	return GetRegistry().ClearAll(p.ctx, characterId)
}

// Delete deletes all skills for a character
func (p *ProcessorImpl) Delete(characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return deleteByCharacter(tx, characterId)
	})
}

// CooldownDecorator returns a decorator that adds cooldown information to a skill model
func (p *ProcessorImpl) CooldownDecorator(characterId uint32) model.Decorator[Model] {
	return model.ErrDecorator(
		func(m Model) (Model, error) {
			ct, err := GetRegistry().Get(p.ctx, characterId, m.Id())
			if err != nil {
				return m, err
			}
			updated, err := CloneModel(m).SetCooldownExpiresAt(ct).Build()
			if err != nil {
				return m, err
			}
			return updated, nil
		},
		func(m Model, err error) {
			degrade.Observe(p.l, "skills.skill.cooldown", characterId, err)
		},
	)
}

// RequestCreate sends a command to create a skill
func (p *ProcessorImpl) RequestCreate(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(createCommandProvider(transactionId, worldId, characterId, id, level, masterLevel, expiration))
}

// RequestUpdate sends a command to update a skill
func (p *ProcessorImpl) RequestUpdate(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(updateCommandProvider(transactionId, worldId, characterId, id, level, masterLevel, expiration))
}

// DeleteForSagaCompensation is the buffer-based inner form: deletes the skill
// row idempotently and buffers a saga-correlated DELETED status event. An
// absent skill is treated as success. See PRD §4.3.1 / §4.8 and plan Phase 5.
func (p *ProcessorImpl) DeleteForSagaCompensation(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error {
		existed, err := deleteSkill(p.db.WithContext(p.ctx), characterId, skillId)
		if err != nil {
			return err
		}
		if !existed {
			p.l.WithFields(logrus.Fields{
				"transaction_id": transactionId.String(),
				"character_id":   characterId,
				"skill_id":       skillId,
			}).Info("Skill already absent; buffering synthetic DELETED event for saga compensation.")
		}
		return mb.Put(skill2.EnvStatusEventTopic, statusEventDeletedProvider(transactionId, worldId, characterId, skillId))
	}
}

// DeleteForSagaCompensationAndEmit wraps DeleteForSagaCompensation with an
// explicit transaction (the underlying deleteSkill write was previously a
// bare, un-transacted call) and the outbox emit flow, so the delete and the
// status-event enqueue commit atomically.
func (p *ProcessorImpl) DeleteForSagaCompensationAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).DeleteForSagaCompensation(buf)(transactionId, worldId, characterId, skillId)
		})
	})
}

// TransferSpAndEmit wraps TransferSp with the producer emit flow.
func (p *ProcessorImpl) TransferSpAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.TransferSp(buf)(transactionId, worldId, characterId, jobId, fromSkillId, toSkillId, itemTier, targetMaxLevel)
	})
}

// TransferSp moves one skill point FromSkillId -> ToSkillId. Structural
// validation (job tree, exclusion list, tier vs. itemTier) happens first and
// needs no DB access; all state-derived validation (source level, target cap)
// and mutation (source -1, target +1, macro cleanup) happens inside ONE
// gorm-native transaction (p.db.Transaction — NOT database.ExecuteTransaction,
// which is a documented no-op) so the skill rows and any macro rows commit or
// roll back together. See task-126 design §4.6.
func (p *ProcessorImpl) TransferSp(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
		reject := func(errType string, detailSkillId uint32) error {
			p.l.WithFields(logrus.Fields{
				"character_id": characterId,
				"from":         fromSkillId,
				"to":           toSkillId,
				"tier":         itemTier,
			}).Warnf("Rejected SP transfer: [%s].", errType)
			_ = mb.Put(skill2.EnvStatusEventTopic, statusEventErrorProvider(transactionId, worldId, characterId, detailSkillId, errType, strconv.FormatUint(uint64(detailSkillId), 10)))
			return nil
		}

		// Structural validation (no DB needed).
		fromJob := job.IdFromSkillId(constskill.Id(fromSkillId))
		toJob := job.IdFromSkillId(constskill.Id(toSkillId))
		if !job.Is(jobId, fromJob) || !job.Is(jobId, toJob) {
			return reject(skill2.StatusEventErrorTypeInvalidTarget, toSkillId)
		}
		if constskill.IsPointResetExcluded(constskill.Id(fromSkillId)) || constskill.IsPointResetExcluded(constskill.Id(toSkillId)) {
			return reject(skill2.StatusEventErrorTypeInvalidTarget, toSkillId)
		}
		fromTier := job.Advancement(fromJob)
		toTier := job.Advancement(toJob)
		if toTier != int(itemTier) || fromTier < 1 || fromTier > int(itemTier) {
			return reject(skill2.StatusEventErrorTypeWrongTier, toSkillId)
		}

		return p.db.Transaction(func(tx *gorm.DB) error {
			sp := p.WithTransaction(tx)

			from, err := sp.ByIdProvider(characterId, fromSkillId)()
			if err != nil || from.Level() == 0 {
				return reject(skill2.StatusEventErrorTypeSkillAtZero, fromSkillId)
			}

			// Target row may not exist yet: treat as level 0 / masterLevel 0.
			var toLevel, toMaster byte
			var toExists bool
			var toExpiration time.Time
			if to, err := sp.ByIdProvider(characterId, toSkillId)(); err == nil {
				toLevel, toMaster, toExpiration, toExists = to.Level(), to.MasterLevel(), to.Expiration(), true
			}

			effectiveCap := targetMaxLevel
			if job.IsFourthJob(toJob) {
				effectiveCap = toMaster // 4th-job cap is the earned master level (design §9.2)
			}
			if toLevel >= effectiveCap {
				return reject(skill2.StatusEventErrorTypeSkillAtCap, toSkillId)
			}

			newFromLevel := from.Level() - 1
			newToLevel := toLevel + 1

			// Apply: source -1, target +1 (master levels untouched, FR-15/16).
			if _, err := sp.Update(mb)(transactionId, worldId, characterId, fromSkillId, newFromLevel, from.MasterLevel(), from.Expiration()); err != nil {
				return err
			}
			if toExists {
				if _, err := sp.Update(mb)(transactionId, worldId, characterId, toSkillId, newToLevel, toMaster, toExpiration); err != nil {
					return err
				}
			} else {
				if _, err := sp.Create(mb)(transactionId, worldId, characterId, toSkillId, newToLevel, 0, time.Time{}); err != nil {
					return err
				}
			}

			// Macro cleanup (FR-18) inside the same tx — only when the source
			// skill drops to level 0.
			if newFromLevel == 0 {
				mp := macro.NewProcessor(p.l, p.ctx, tx)
				macros, err := mp.ByCharacterIdProvider(characterId)()
				if err != nil {
					return err
				}
				changed := false
				updated := make([]macro.Model, 0, len(macros))
				for _, m := range macros {
					b := macro.CloneModel(m)
					if uint32(m.SkillId1()) == fromSkillId {
						b = b.SetSkillId1(0)
						changed = true
					}
					if uint32(m.SkillId2()) == fromSkillId {
						b = b.SetSkillId2(0)
						changed = true
					}
					if uint32(m.SkillId3()) == fromSkillId {
						b = b.SetSkillId3(0)
						changed = true
					}
					nm, err := b.Build()
					if err != nil {
						return err
					}
					updated = append(updated, nm)
				}
				if changed {
					if _, err := mp.Update(mb)(transactionId, worldId, characterId, updated); err != nil {
						return err
					}
				}
			}

			_ = mb.Put(skill2.EnvStatusEventTopic, statusEventSpTransferredProvider(transactionId, worldId, characterId, toSkillId, fromSkillId, newFromLevel, newToLevel))
			return nil
		})
	}
}
