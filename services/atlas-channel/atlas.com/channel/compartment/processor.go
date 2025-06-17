package compartment

import (
	"atlas-channel/kafka/message/compartment"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
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

func (p *Processor) Unequip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(UnequipAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *Processor) Equip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(EquipAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *Processor) Move(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(MoveAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *Processor) Drop(m _map.Model, characterId uint32, inventoryType inventory.Type, source int16, quantity int16, x int16, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(DropAssetCommandProvider(m, characterId, inventoryType, source, quantity, x, y))
}

func (p *Processor) Merge(characterId uint32, inventoryType inventory.Type, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to merge compartment [%d]. updateTime [%d].", characterId, inventoryType, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(MergeCommandProvider(characterId, inventoryType))
}

func (p *Processor) Sort(characterId uint32, inventoryType inventory.Type, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to sort compartment [%d]. updateTime [%d].", characterId, inventoryType, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(SortCommandProvider(characterId, inventoryType))
}

func (p *Processor) MoveToOtherInventory(characterId uint32, inventoryType inventory.Type, slot int16, otherInventory string) error {
	p.l.Debugf("Character [%d] attempting to move item in [%d] slot [%d] to [%s] inventory.", characterId, inventoryType, slot, otherInventory)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(MoveToCommandProvider(characterId, inventoryType, slot, otherInventory))
}
