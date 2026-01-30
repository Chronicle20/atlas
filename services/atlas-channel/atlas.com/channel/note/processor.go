package note

import (
	note2 "atlas-channel/kafka/message/note"
	"atlas-channel/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for note processing
type Processor interface {
	ByCharacterProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacter(characterId uint32) ([]Model, error)
	ByIdProvider(noteId uint32) model.Provider[Model]
	GetById(noteId uint32) (Model, error)
	SendNote(worldId world.Id, channelId channel.Id, senderId uint32, receiverId uint32, message string, flag byte) error
	DiscardNotes(worldId world.Id, channelId channel.Id, characterId uint32, noteIds []uint32) error
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

func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
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

func (p *ProcessorImpl) SendNote(worldId world.Id, channelId channel.Id, senderId uint32, receiverId uint32, message string, flag byte) error {
	p.l.Debugf("Character [%d] attempting to send note to [%d].", senderId, receiverId)
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(CreateCommandProvider(worldId, channelId, senderId, receiverId, message, flag))
}

func (p *ProcessorImpl) DiscardNotes(worldId world.Id, channelId channel.Id, characterId uint32, noteIds []uint32) error {
	if len(noteIds) == 0 {
		return errors.New("no note IDs provided")
	}
	p.l.Debugf("Character [%d] attempting to discard [%d] notes.", characterId, len(noteIds))
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(DiscardCommandProvider(worldId, channelId, characterId, noteIds))
}
