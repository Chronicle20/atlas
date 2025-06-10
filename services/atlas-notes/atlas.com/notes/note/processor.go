package note

import (
	"atlas-notes/kafka/message"
	"atlas-notes/kafka/message/note"
	"atlas-notes/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
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
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		producer: producer.ProviderImpl(l)(ctx),
	}
}

// Create creates a new note
func (p *ProcessorImpl) Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
	return func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
		return func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
			return func(msg string) func(flag byte) (Model, error) {
				return func(flag byte) (Model, error) {
					m := NewBuilder().
						SetCharacterId(characterId).
						SetSenderId(senderId).
						SetMessage(msg).
						SetFlag(flag).
						Build()

					m, err := createNote(p.db)(p.t.Id())(m)
					if err != nil {
						return Model{}, err
					}
					err = mb.Put(note.EnvEventTopicNoteStatus, CreateNoteStatusEvent(m.CharacterId(), m.Id(), m.SenderId(), m.Message(), m.Flag(), m.Timestamp()))
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
						m := NewBuilder().
							SetId(id).
							SetCharacterId(characterId).
							SetSenderId(senderId).
							SetMessage(msg).
							SetFlag(flag).
							Build()

						m, err := updateNote(p.db)(p.t.Id())(m)
						if err != nil {
							return Model{}, err
						}
						err = mb.Put(note.EnvEventTopicNoteStatus, UpdateNoteStatusEvent(m.CharacterId(), m.Id(), m.SenderId(), m.Message(), m.Flag(), m.Timestamp()))
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
		err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEvent(m.CharacterId(), id))
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
			err = mb.Put(note.EnvEventTopicNoteStatus, DeleteNoteStatusEvent(m.CharacterId(), m.Id()))
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
