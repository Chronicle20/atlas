package note

import (
	"atlas-notes/kafka/message"
	"atlas-notes/kafka/message/note"
	"atlas-notes/saga"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)
	CreateAndEmit(characterId uint32, senderId uint32, msg string, flag byte) (Model, error)
	Update(mb *message.Buffer) func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)
	UpdateAndEmit(id uint32, characterId uint32, senderId uint32, msg string, flag byte) (Model, error)
	Delete(mb *message.Buffer) func(id uint32) error
	DeleteAndEmit(id uint32) error
	DeleteAll(mb *message.Buffer) func(characterId uint32) error
	DeleteAllAndEmit(characterId uint32) error
	Discard(mb *message.Buffer) func(ch channel.Model) func(characterId uint32) func(noteIds []uint32) ([]pendingFameAward, error)
	DiscardAndEmit(ch channel.Model, characterId uint32, noteIds []uint32) error
	ByIdProvider(id uint32) model.Provider[Model]
	ByCharacterProvider(characterId uint32) model.Provider[[]Model]
	InTenantProvider() model.Provider[[]Model]
}

type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	db    *gorm.DB
	t     tenant.Model
	sagaP saga.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		sagaP: saga.NewProcessor(l, ctx),
	}
}

// WithTransaction returns a copy of the processor bound to the given transaction, so
// nested administrator writes and the outbox enqueue for a migrated *AndEmit method
// join the same transaction.
func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:     p.l,
		ctx:   p.ctx,
		db:    tx,
		t:     p.t,
		sagaP: p.sagaP,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create creates a new note
func (p *ProcessorImpl) Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
	return func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
		return func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
			return func(msg string) func(flag byte) (Model, error) {
				return func(flag byte) (Model, error) {
					m, err := NewBuilder().
						SetCharacterId(characterId).
						SetSenderId(senderId).
						SetMessage(msg).
						SetFlag(flag).
						Build()
					if err != nil {
						return Model{}, err
					}

					m, err = createNote(p.db.WithContext(p.ctx), p.t.Id(), m)
					if err != nil {
						return Model{}, err
					}
					err = mb.Put(note.EnvEventTopicNoteStatus, CreateNoteStatusEventProvider(m.CharacterId(), m.Id(), m.SenderId(), m.Message(), m.Flag(), m.Timestamp()))
					if err != nil {
						return Model{}, err
					}
					return m, nil
				}
			}
		}
	}
}

// CreateAndEmit creates a new note and emits a status event
func (p *ProcessorImpl) CreateAndEmit(characterId uint32, senderId uint32, msg string, flag byte) (Model, error) {
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		tp := p.WithTransaction(tx)
		result, err = message.EmitWithResult[Model, byte](outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(model.Flip(model.Flip(tp.Create)(characterId))(senderId))(msg))(flag)
		return err
	})
	return result, txErr
}

// Update updates an existing note
func (p *ProcessorImpl) Update(mb *message.Buffer) func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
	return func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
		return func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
			return func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
				return func(msg string) func(flag byte) (Model, error) {
					return func(flag byte) (Model, error) {
						m, err := NewBuilder().
							SetId(id).
							SetCharacterId(characterId).
							SetSenderId(senderId).
							SetMessage(msg).
							SetFlag(flag).
							Build()
						if err != nil {
							return Model{}, err
						}

						m, err = updateNote(p.db.WithContext(p.ctx), p.t.Id(), m)
						if err != nil {
							return Model{}, err
						}
						err = mb.Put(note.EnvEventTopicNoteStatus, UpdateNoteStatusEventProvider(m.CharacterId(), m.Id(), m.SenderId(), m.Message(), m.Flag(), m.Timestamp()))
						if err != nil {
							return Model{}, err
						}
						return m, nil
					}
				}
			}
		}
	}
}

// UpdateAndEmit updates an existing note and emits a status event
func (p *ProcessorImpl) UpdateAndEmit(id uint32, characterId uint32, senderId uint32, msg string, flag byte) (Model, error) {
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		tp := p.WithTransaction(tx)
		result, err = message.EmitWithResult[Model, byte](outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(model.Flip(model.Flip(model.Flip(tp.Update)(id))(characterId))(senderId))(msg))(flag)
		return err
	})
	return result, txErr
}

// Delete deletes a note
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(id uint32) error {
	return func(id uint32) error {
		m, err := p.ByIdProvider(id)()
		if err != nil {
			return err
		}

		err = deleteNote(p.db.WithContext(p.ctx), id)
		if err != nil {
			return err
		}
		err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEventProvider(m.CharacterId(), id))
		if err != nil {
			return err
		}
		return nil
	}
}

// DeleteAndEmit deletes a note and emits a status event
func (p *ProcessorImpl) DeleteAndEmit(id uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		tp := p.WithTransaction(tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(tp.Delete)(id))
	})
}

// DeleteAll deletes all notes for a character
func (p *ProcessorImpl) DeleteAll(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		ms, err := p.ByCharacterProvider(characterId)()
		if err != nil {
			return err
		}
		for _, m := range ms {
			err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEventProvider(m.CharacterId(), m.Id()))
			if err != nil {
				return err
			}
		}
		err = deleteAllNotes(p.db.WithContext(p.ctx), characterId)
		if err != nil {
			return err
		}
		return nil
	}
}

// DeleteAllAndEmit deletes all notes for a character and emits status events
func (p *ProcessorImpl) DeleteAllAndEmit(characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		tp := p.WithTransaction(tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(tp.DeleteAll)(characterId))
	})
}

// ByIdProvider retrieves a note by ID
func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByCharacterProvider retrieves all notes for a character
func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getByCharacterIdProvider(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// InTenantProvider retrieves all notes in a tenant
func (p *ProcessorImpl) InTenantProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider()(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// pendingFameAward pairs a built fame-award saga with the note/sender ids used for logging when it
// is later fired. It is collected during Discard's per-note loop (which runs inside the shared
// discard transaction) but must NOT be fired until that transaction has committed successfully -
// firing mid-loop would send an unrecallable saga command for a delete that a later note's failure
// could still roll back.
type pendingFameAward struct {
	saga     saga.Saga
	noteId   uint32
	senderId uint32
}

// Discard discards multiple notes for a character. It collects (but does not fire) the fame-award
// saga command for each successfully discarded note; the caller is responsible for firing them only
// after the enclosing transaction has committed (see DiscardAndEmit).
func (p *ProcessorImpl) Discard(mb *message.Buffer) func(ch channel.Model) func(characterId uint32) func(noteIds []uint32) ([]pendingFameAward, error) {
	return func(ch channel.Model) func(characterId uint32) func(noteIds []uint32) ([]pendingFameAward, error) {
		return func(characterId uint32) func(noteIds []uint32) ([]pendingFameAward, error) {
			return func(noteIds []uint32) ([]pendingFameAward, error) {
				var pending []pendingFameAward
				for _, noteId := range noteIds {
					// Check if the note exists and belongs to the character
					m, err := p.ByIdProvider(noteId)()
					if err != nil {
						return nil, err
					}

					if m.CharacterId() != characterId {
						continue // Skip notes that don't belong to this character
					}

					// Delete the note
					err = deleteNote(p.db.WithContext(p.ctx), noteId)
					if err != nil {
						return nil, err
					}

					// Add delete event to message buffer
					err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEventProvider(characterId, noteId))
					if err != nil {
						return nil, err
					}

					// Build (but do not fire) the fame-award saga for the note sender
					if pa, ok := p.buildFameAwardSaga(ch, characterId, m.SenderId(), noteId); ok {
						pending = append(pending, pa)
					}
				}
				return pending, nil
			}
		}
	}
}

// buildFameAwardSaga builds (without firing) the saga to award +1 fame to a note sender. It returns
// ok=false when the award should be skipped (system note or self-note). This is a pure builder with
// no side effects, so it is safe to call from inside a transaction closure; firing it is a separate
// step performed by fireFameAwardSaga.
func (p *ProcessorImpl) buildFameAwardSaga(ch channel.Model, recipientId uint32, senderId uint32, noteId uint32) (pendingFameAward, bool) {
	// Skip if sender is 0 (system note) or sender is the same as recipient (self-note)
	if senderId == 0 {
		p.l.Debugf("Skipping fame award for note [%d]: system note (senderId=0)", noteId)
		return pendingFameAward{}, false
	}
	if senderId == recipientId {
		p.l.Debugf("Skipping fame award for note [%d]: self-note (senderId=%d equals recipientId)", noteId, senderId)
		return pendingFameAward{}, false
	}

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("note-discard-fame").
		AddStep(
			fmt.Sprintf("award-fame-%d-%d", senderId, noteId),
			saga.Pending,
			saga.AwardFame,
			saga.AwardFamePayload{
				CharacterId: senderId,
				WorldId:     ch.WorldId(),
				ChannelId:   ch.Id(),
				Amount:      1,
			},
		).Build()

	return pendingFameAward{saga: s, noteId: noteId, senderId: senderId}, true
}

// fireFameAwardSaga sends a previously built fame-award saga command to atlas-saga-orchestrator.
// Errors are logged but do not fail the discard operation, matching prior behavior.
func (p *ProcessorImpl) fireFameAwardSaga(pa pendingFameAward) {
	err := p.sagaP.Create(pa.saga)
	if err != nil {
		// Log error but don't fail the discard operation
		p.l.WithError(err).Errorf("Failed to create fame award saga for note [%d] sender [%d]", pa.noteId, pa.senderId)
	} else {
		p.l.Debugf("Created fame award saga for note [%d] sender [%d]", pa.noteId, pa.senderId)
	}
}

// DiscardAndEmit discards multiple notes for a character and emits status events. The fame-award
// saga commands collected during the discard are fired only after the transaction has committed
// successfully, so a rolled-back delete can never leave behind an already-fired, unrecallable fame
// award (see task-114 review: ExecuteTransaction wraps the whole loop in one shared tx).
func (p *ProcessorImpl) DiscardAndEmit(ch channel.Model, characterId uint32, noteIds []uint32) error {
	var pending []pendingFameAward
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		tp := p.WithTransaction(tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			var err error
			pending, err = tp.Discard(mb)(ch)(characterId)(noteIds)
			return err
		})
	})
	if txErr != nil {
		return txErr
	}
	for _, pa := range pending {
		p.fireFameAwardSaga(pa)
	}
	return nil
}
