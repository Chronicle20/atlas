package note

import (
	note2 "atlas-channel/kafka/message/note"
	"context"
	"errors"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for note processing
type Processor interface {
	ByCharacterProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacter(characterId uint32) ([]Model, error)
	ByIdProvider(noteId uint32) model.Provider[Model]
	GetById(noteId uint32) (Model, error)
	SendNote(ch channel.Model, senderId uint32, receiverId uint32, message string, flag byte) error
	DiscardNotes(ch channel.Model, characterId uint32, noteIds []uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)


// ByCharacterProvider fetches the complete set of notes for a character.
// The upstream atlas-notes list is now paginated (task-117); callers here
// need the whole set (e.g. delivering all pending notes on login), so this
// drains every page rather than fetching one.
func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterNotesUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return p.ByCharacterProvider(characterId)()
}

func (p *ProcessorImpl) ByIdProvider(noteId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(noteId), Extract)
}

func (p *ProcessorImpl) GetById(noteId uint32) (Model, error) {
	return p.ByIdProvider(noteId)()
}

func (p *ProcessorImpl) SendNote(ch channel.Model, senderId uint32, receiverId uint32, message string, flag byte) error {
	p.l.Debugf("Character [%d] attempting to send note to [%d].", senderId, receiverId)
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(CreateCommandProvider(ch, senderId, receiverId, message, flag))
}

func (p *ProcessorImpl) DiscardNotes(ch channel.Model, characterId uint32, noteIds []uint32) error {
	if len(noteIds) == 0 {
		return errors.New("no note IDs provided")
	}
	p.l.Debugf("Character [%d] attempting to discard [%d] notes.", characterId, len(noteIds))
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(DiscardCommandProvider(ch, characterId, noteIds))
}
