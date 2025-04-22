package inventory

import (
	inventory2 "atlas-channel/kafka/message/inventory"
	"atlas-channel/kafka/producer"
	inventory3 "atlas-channel/kafka/producer/inventory"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func (p *Processor) Unequip(characterId uint32, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(inventory2.EnvCommandTopic)(inventory3.UnequipItemCommandProvider(characterId, source, destination))
}

func (p *Processor) Equip(characterId uint32, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(inventory2.EnvCommandTopic)(inventory3.EquipItemCommandProvider(characterId, source, destination))
}

func (p *Processor) Move(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(inventory2.EnvCommandTopic)(inventory3.MoveItemCommandProvider(characterId, inventoryType, source, destination))
}

func (p *Processor) Drop(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(inventory2.EnvCommandTopic)(inventory3.DropItemCommandProvider(m, characterId, inventoryType, source, quantity))
}
