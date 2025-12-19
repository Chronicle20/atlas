package messenger

import (
	messenger2 "atlas-channel/kafka/message/messenger"
	"atlas-channel/kafka/producer"
	"context"
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

func (p *Processor) Create(characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to create a messenger.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(messenger2.EnvCommandTopic)(CreateCommandProvider(characterId))
}

func (p *Processor) Leave(messengerId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to leave messenger [%d].", characterId, messengerId)
	return producer.ProviderImpl(p.l)(p.ctx)(messenger2.EnvCommandTopic)(LeaveCommandProvider(characterId, messengerId))
}

func (p *Processor) RequestInvite(characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to invite [%d] to a messenger.", characterId, targetCharacterId)
	return producer.ProviderImpl(p.l)(p.ctx)(messenger2.EnvCommandTopic)(RequestInviteCommandProvider(characterId, targetCharacterId))
}

func (p *Processor) GetById(messengerId uint32) (Model, error) {
	return p.ByIdProvider(messengerId)()
}

func (p *Processor) ByIdProvider(messengerId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(messengerId), Extract)
}

func (p *Processor) GetByMemberId(memberId uint32) (Model, error) {
	return p.ByMemberIdProvider(memberId)()
}

func (p *Processor) ByMemberIdProvider(memberId uint32) model.Provider[Model] {
	rp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
	return model.FirstProvider(rp, model.Filters[Model]())
}
