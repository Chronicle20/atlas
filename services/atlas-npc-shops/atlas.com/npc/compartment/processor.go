package compartment

import (
	"atlas-npc/kafka/message/compartment"
	"atlas-npc/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestCreateItem(characterId uint32, templateId uint32, quantity uint32) error
	RequestDestroyItem(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	RequestRechargeItem(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
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

func (p *ProcessorImpl) RequestCreateItem(characterId uint32, templateId uint32, quantity uint32) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestCreateAssetCommandProvider(characterId, inventoryType, templateId, quantity))
}

func (p *ProcessorImpl) RequestDestroyItem(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestDestroyAssetCommandProvider(characterId, inventoryType, slot, quantity))
}

func (p *ProcessorImpl) RequestRechargeItem(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestRechargeAssetCommandProvider(characterId, inventoryType, slot, quantity))
}
