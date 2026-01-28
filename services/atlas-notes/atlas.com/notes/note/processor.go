package note

import (
	"atlas-notes/kafka/message"
	"atlas-notes/kafka/message/note"
	"atlas-notes/kafka/producer"
	"atlas-notes/saga"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	scriptsaga "github.com/Chronicle20/atlas-script-core/saga"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)
	CreateAndEmit(characterId uint32, senderId uint32, msg string, flag byte) (Model, error)
	Update(mb *message.Buffer) func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)
	UpdateAndEmit(id uint32, characterId uint32, senderId uint32, msg string, flag byte) (Model, error)
	Delete(mb *message.Buffer) func(id uint32) error
	DeleteAndEmit(id uint32) error
	DeleteAll(mb *message.Buffer) func(characterId uint32) error
	DeleteAllAndEmit(characterId uint32) error
	Discard(mb *message.Buffer) func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(noteIds []uint32) error
	DiscardAndEmit(worldId world.Id, channelId channel.Id, characterId uint32, noteIds []uint32) error
	ByIdProvider(id uint32) model.Provider[Model]
	ByCharacterProvider(characterId uint32) model.Provider[[]Model]
	InTenantProvider() model.Provider[[]Model]
}

type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	t        tenant.Model
	producer producer.Provider
	sagaP    saga.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		producer: producer.ProviderImpl(l)(ctx),
		sagaP:    saga.NewProcessor(l, ctx),
	}
}

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

					m, err = createNote(p.db)(p.t.Id())(m)
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
	return message.EmitWithResult[Model, byte](p.producer)(model.Flip(model.Flip(model.Flip(p.Create)(characterId))(senderId))(msg))(flag)
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

						m, err = updateNote(p.db)(p.t.Id())(m)
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
	return message.EmitWithResult[Model, byte](p.producer)(model.Flip(model.Flip(model.Flip(model.Flip(p.Update)(id))(characterId))(senderId))(msg))(flag)
}

// Delete deletes a note
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(id uint32) error {
	return func(id uint32) error {
		m, err := p.ByIdProvider(id)()
		if err != nil {
			return err
		}

		err = deleteNote(p.db)(p.t.Id())(id)
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
	return message.Emit(p.producer)(model.Flip(p.Delete)(id))
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
		err = deleteAllNotes(p.db)(p.t.Id())(characterId)
		if err != nil {
			return err
		}
		return nil
	}
}

// DeleteAllAndEmit deletes all notes for a character and emits status events
func (p *ProcessorImpl) DeleteAllAndEmit(characterId uint32) error {
	return message.Emit(p.producer)(model.Flip(p.DeleteAll)(characterId))
}

// ByIdProvider retrieves a note by ID
func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

// ByCharacterProvider retrieves all notes for a character
func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getByCharacterIdProvider(p.t.Id())(characterId)(p.db))(model.ParallelMap())
}

// InTenantProvider retrieves all notes in a tenant
func (p *ProcessorImpl) InTenantProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

// Discard discards multiple notes for a character
func (p *ProcessorImpl) Discard(mb *message.Buffer) func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(noteIds []uint32) error {
	return func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(noteIds []uint32) error {
		return func(channelId channel.Id) func(characterId uint32) func(noteIds []uint32) error {
			return func(characterId uint32) func(noteIds []uint32) error {
				return func(noteIds []uint32) error {
					for _, noteId := range noteIds {
						// Check if the note exists and belongs to the character
						m, err := p.ByIdProvider(noteId)()
						if err != nil {
							return err
						}

						if m.CharacterId() != characterId {
							continue // Skip notes that don't belong to this character
						}

						// Delete the note
						err = deleteNote(p.db)(p.t.Id())(noteId)
						if err != nil {
							return err
						}

						// Add delete event to message buffer
						err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEventProvider(characterId, noteId))
						if err != nil {
							return err
						}

						// Award fame to the note sender
						p.awardFameToSender(worldId, channelId, characterId, m.SenderId(), noteId)
					}
					return nil
				}
			}
		}
	}
}

// awardFameToSender creates a saga to award +1 fame to the note sender
func (p *ProcessorImpl) awardFameToSender(worldId world.Id, channelId channel.Id, recipientId uint32, senderId uint32, noteId uint32) {
	// Skip if sender is 0 (system note) or sender is the same as recipient (self-note)
	if senderId == 0 {
		p.l.Debugf("Skipping fame award for note [%d]: system note (senderId=0)", noteId)
		return
	}
	if senderId == recipientId {
		p.l.Debugf("Skipping fame award for note [%d]: self-note (senderId=%d equals recipientId)", noteId, senderId)
		return
	}

	s := scriptsaga.NewBuilder().
		SetSagaType(scriptsaga.InventoryTransaction).
		SetInitiatedBy("note-discard-fame").
		AddStep(
			fmt.Sprintf("award-fame-%d-%d", senderId, noteId),
			scriptsaga.Pending,
			scriptsaga.AwardFame,
			scriptsaga.AwardFamePayload{
				CharacterId: senderId,
				WorldId:     worldId,
				ChannelId:   channelId,
				Amount:      1,
			},
		).Build()

	err := p.sagaP.Create(s)
	if err != nil {
		// Log error but don't fail the discard operation
		p.l.WithError(err).Errorf("Failed to create fame award saga for note [%d] sender [%d]", noteId, senderId)
	} else {
		p.l.Debugf("Created fame award saga for note [%d] sender [%d]", noteId, senderId)
	}
}

// DiscardAndEmit discards multiple notes for a character and emits status events
func (p *ProcessorImpl) DiscardAndEmit(worldId world.Id, channelId channel.Id, characterId uint32, noteIds []uint32) error {
	return message.Emit(p.producer)(func(mb *message.Buffer) error {
		return p.Discard(mb)(worldId)(channelId)(characterId)(noteIds)
	})
}
