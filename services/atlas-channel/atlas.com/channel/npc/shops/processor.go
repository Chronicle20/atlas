package shops

import (
	shops2 "atlas-channel/kafka/message/npc/shop"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByTemplateIdProvider(templateId uint32) model.Provider[Model]
	GetShop(template uint32) (Model, error)
	EnterShop(characterId uint32, npcTemplateId uint32) error
	ExitShop(characterId uint32) error
}

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

func (p *ProcessorImpl) ByTemplateIdProvider(templateId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestNPCShop(templateId), Extract)
}

func (p *ProcessorImpl) GetShop(template uint32) (Model, error) {
	return p.ByTemplateIdProvider(template)()
}

func (p *ProcessorImpl) EnterShop(characterId uint32, npcTemplateId uint32) error {
	p.l.Debugf("Character [%d] is entering NPC shop [%d].", characterId, npcTemplateId)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopEnterCommandProvider(characterId, npcTemplateId))
}

func (p *ProcessorImpl) ExitShop(characterId uint32) error {
	p.l.Debugf("Character [%d] is exiting NPC shop.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopExitCommandProvider(characterId))
}
