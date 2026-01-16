package compartment

import (
	"atlas-channel/kafka/message/compartment"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model]
	GetByType(characterId uint32, inventoryType inventory.Type) (Model, error)
	Unequip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	Equip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	Move(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	Drop(m _map.Model, characterId uint32, inventoryType inventory.Type, source int16, quantity int16, x int16, y int16) error
	Merge(characterId uint32, inventoryType inventory.Type, updateTime uint32) error
	Sort(characterId uint32, inventoryType inventory.Type, updateTime uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByType(characterId, inventoryType), Extract)
}

func (p *ProcessorImpl) GetByType(characterId uint32, inventoryType inventory.Type) (Model, error) {
	return p.ByCharacterIdAndTypeProvider(characterId, inventoryType)()
}

func (p *ProcessorImpl) Unequip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(UnequipAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) Equip(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(EquipAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) Move(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(MoveAssetCommandProvider(characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) Drop(m _map.Model, characterId uint32, inventoryType inventory.Type, source int16, quantity int16, x int16, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(DropAssetCommandProvider(m, characterId, inventoryType, source, quantity, x, y))
}

func (p *ProcessorImpl) Merge(characterId uint32, inventoryType inventory.Type, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to merge compartment [%d]. updateTime [%d].", characterId, inventoryType, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(MergeCommandProvider(characterId, inventoryType))
}

func (p *ProcessorImpl) Sort(characterId uint32, inventoryType inventory.Type, updateTime uint32) error {
	p.l.Debugf("Character [%d] attempting to sort compartment [%d]. updateTime [%d].", characterId, inventoryType, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(SortCommandProvider(characterId, inventoryType))
}
