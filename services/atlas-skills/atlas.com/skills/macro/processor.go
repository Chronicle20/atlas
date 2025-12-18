package macro

import (
	"atlas-skills/database"
	"atlas-skills/kafka/message"
	macro2 "atlas-skills/kafka/message/macro"
	"atlas-skills/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor defines the interface for macro processing operations
type Processor interface {
	// ByCharacterIdProvider returns a provider for all macros for a character
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]

	// Update updates all macros for a character with message buffer for events
	Update(mb *message.Buffer) func(characterId uint32) func(macros []Model) ([]Model, error)

	// UpdateAndEmit updates all macros for a character and emits events
	UpdateAndEmit(characterId uint32, macros []Model) ([]Model, error)

	// Delete deletes all macros for a character
	Delete(characterId uint32) error
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

// ByCharacterIdProvider returns a provider for all macros for a character
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByCharacterId(p.t.Id(), characterId)(p.db))(model.ParallelMap())
}

// Update updates all macros for a character with message buffer for events
func (p *ProcessorImpl) Update(mb *message.Buffer) func(characterId uint32) func(macros []Model) ([]Model, error) {
	return func(characterId uint32) func(macros []Model) ([]Model, error) {
		return func(macros []Model) ([]Model, error) {
			p.l.Debugf("Updating skill macros for character [%d].", characterId)
			var result []Model

			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
				err := deleteByCharacter(tx, p.t, characterId)
				if err != nil {
					return err
				}
				for _, macro := range macros {
					_, err = create(tx, p.t.Id(), macro.Id(), characterId, macro.Name(), macro.Shout(), uint32(macro.SkillId1()), uint32(macro.SkillId2()), uint32(macro.SkillId3()))()
					if err != nil {
						return err
					}
				}
				return nil
			})

			if txErr != nil {
				return nil, txErr
			}

			// Get the updated macros
			result, err := p.ByCharacterIdProvider(characterId)()
			if err != nil {
				return nil, err
			}

			// Add the status event to the message buffer
			_ = mb.Put(macro2.EnvStatusEventTopic, statusEventUpdatedProvider(characterId, result))

			return result, nil
		}
	}
}

// UpdateAndEmit updates all macros for a character and emits events
func (p *ProcessorImpl) UpdateAndEmit(characterId uint32, macros []Model) ([]Model, error) {
	var result []Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		result, err = p.Update(buf)(characterId)(macros)
		return err
	})
	return result, err
}

// Delete deletes all macros for a character
func (p *ProcessorImpl) Delete(characterId uint32) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return deleteByCharacter(tx, p.t, characterId)
	})
}
