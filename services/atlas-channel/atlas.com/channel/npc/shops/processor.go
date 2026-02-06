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
	BuyItem(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error
	SellItem(characterId uint32, slot int16, itemTemplateId uint32, quantity uint32) error
	RechargeItem(characterId uint32, slot uint16) error
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

func (p *ProcessorImpl) BuyItem(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error {
	p.l.Debugf("Character [%d] is buying [%d] item [%d] from slot [%d] in NPC shop at price [%d].", characterId, quantity, itemTemplateId, slot, discountPrice)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopBuyCommandProvider(characterId, slot, itemTemplateId, quantity, discountPrice))
}

func (p *ProcessorImpl) SellItem(characterId uint32, slot int16, itemTemplateId uint32, quantity uint32) error {
	p.l.Debugf("Character [%d] is selling [%d] item [%d] from slot [%d] in NPC shop.", characterId, quantity, itemTemplateId, slot)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopSellCommandProvider(characterId, slot, itemTemplateId, quantity))
}

func (p *ProcessorImpl) RechargeItem(characterId uint32, slot uint16) error {
	p.l.Debugf("Character [%d] is recharging slot [%d] in NPC shop.", characterId, slot)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopRechargeCommandProvider(characterId, slot))
}
