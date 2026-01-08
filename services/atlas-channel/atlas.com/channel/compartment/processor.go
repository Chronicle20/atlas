package compartment

import (
	"atlas-channel/kafka/message/compartment"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
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

func (p *ProcessorImpl) Transfer(accountId uint32, characterId uint32, assetId uint32, fromId uuid.UUID, fromType byte, fromInventoryType string, toId uuid.UUID, toType byte, toInventoryType string, referenceId uint32) error {
	p.l.Debugf("Character [%d] attempting to transfer asset [%d] from [%s] to [%s] inventory.", characterId, assetId, fromId, toId)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopicCompartmentTransfer)(TransferProvider(accountId, characterId, assetId, fromId, fromType, fromInventoryType, toId, toType, toInventoryType, referenceId))
}

// TransferToStorage transfers an asset from character inventory to storage
func (p *ProcessorImpl) TransferToStorage(worldId byte, accountId uint32, characterId uint32, assetId uint32, fromId uuid.UUID, fromType byte, referenceId uint32, templateId uint32, referenceType string, slot int16) error {
	p.l.Debugf("Character [%d] attempting to transfer asset [%d] to storage.", characterId, assetId)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopicCompartmentTransfer)(StorageTransferProvider(
		worldId, accountId, characterId, assetId,
		fromId, fromType, compartment.InventoryTypeCharacter,
		uuid.Nil, 0, compartment.InventoryTypeStorage,
		referenceId, templateId, referenceType, slot,
	))
}

// TransferFromStorage transfers an asset from storage to character inventory
func (p *ProcessorImpl) TransferFromStorage(worldId byte, accountId uint32, characterId uint32, assetId uint32, toId uuid.UUID, toType byte, referenceId uint32) error {
	p.l.Debugf("Character [%d] attempting to transfer asset [%d] from storage.", characterId, assetId)
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopicCompartmentTransfer)(StorageTransferProvider(
		worldId, accountId, characterId, assetId,
		uuid.Nil, 0, compartment.InventoryTypeStorage,
		toId, toType, compartment.InventoryTypeCharacter,
		referenceId, 0, "", 0,
	))
}
