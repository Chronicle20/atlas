package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ItemPayload represents an individual item in a transaction
type ItemPayload struct {
	TemplateId uint32 `json:"templateId"` // TemplateId of the item
	Quantity   uint32 `json:"quantity"`   // Quantity of the item
}

// CreateAndEquipAssetPayload represents the payload required to create and equip an asset
type CreateAndEquipAssetPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	Item        ItemPayload `json:"item"`        // Item to create and equip
}

type Processor interface {
	RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error
	RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error
	RequestEquipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestUnequipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestCreateAndEquipAsset(transactionId uuid.UUID, payload CreateAndEquipAssetPayload) error
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

func (p *ProcessorImpl) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity))
}

func (p *ProcessorImpl) RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}

	// TODO: Perform transformation from templateId and quantity to slot and quantity
	// The compartment kafka command requires slot and quantity, but we're receiving templateId and quantity
	// This will require looking up the item in the character's inventory to find the slot

	// For now, we'll use a placeholder slot value of -1
	slot := int16(-1)

	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestDestroyAssetCommandProvider(transactionId, characterId, inventoryType, slot, quantity))
}

func (p *ProcessorImpl) RequestEquipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestEquipAssetCommandProvider(transactionId, characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) RequestUnequipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestUnequipAssetCommandProvider(transactionId, characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) RequestCreateAndEquipAsset(transactionId uuid.UUID, payload CreateAndEquipAssetPayload) error {
	// This method internally uses the same award_asset semantics as RequestCreateItem
	// The subsequent equip_asset step will be dynamically created by the compartment consumer
	// when it receives the StatusEventTypeCreated event
	return p.RequestCreateItem(transactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)
}
