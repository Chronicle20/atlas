package note

import (
	note2 "atlas-channel/kafka/message/note"
	"atlas-channel/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByCharacter(characterId uint32) ([]Model, error) {
	return p.ByCharacterProvider(characterId)()
}

func (p *Processor) ByIdProvider(noteId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(noteId), Extract)
}

func (p *Processor) GetById(noteId uint32) (Model, error) {
	return p.ByIdProvider(noteId)()
}

func (p *Processor) SendNote(senderId uint32, receiverId uint32, message string, flag byte) error {
	p.l.Debugf("Character [%d] attempting to send note to [%d].", senderId, receiverId)
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(CreateCommandProvider(senderId, receiverId, message, flag))
}

func (p *Processor) DiscardNotes(characterId uint32, noteIds []uint32) error {
	if len(noteIds) == 0 {
		return errors.New("no note IDs provided")
	}
	p.l.Debugf("Character [%d] attempting to discard [%d] notes.", characterId, len(noteIds))
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(DiscardCommandProvider(characterId, noteIds))
}
